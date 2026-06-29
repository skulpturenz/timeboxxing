package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"strings"

	grpcLogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	grpcAma "github.com/skulpturenz/timeboxxing/sidecar/grpc/ama"
	grpcSettings "github.com/skulpturenz/timeboxxing/sidecar/grpc/settings"
	grpcTransitions "github.com/skulpturenz/timeboxxing/sidecar/grpc/transitions"
	grpcUsage "github.com/skulpturenz/timeboxxing/sidecar/grpc/usage"
	sidecarLogging "github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/workers"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	component                    = "grpc-example"
	startupSemanticBackfillLimit = int64(1000)
)

func Run(ctx context.Context, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	registry := services.New()
	sidecarLogging.RegisterLogger(registry, logger)

	database, err := db.New(ctx, db.Options{
		Engine:         envs.DatabaseEngine.Value(),
		DataSourceName: envs.DatabaseDSN.Value(),
	})
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer database.Close()
	db.Register(registry, database)

	if err := buildQueues(ctx, registry); err != nil {
		return err
	}

	semanticRuntime := buildSemanticRuntime(ctx, registry, database, logger)
	componentTransitions.NewService(registry)
	workerRuntime := workers.NewRuntime(registry)
	for _, cleanup := range startWorkers(ctx, registry, workerRuntime, semanticRuntime) {
		defer cleanup()
	}

	monitorHandle, err := monitor.Start(ctx, logger.With("service", "monitor"), monitor.Config{})
	if err != nil {
		return fmt.Errorf("start monitor: %w", err)
	}
	componentUsage.RegisterActiveSessions(registry, monitorHandle)
	usageService := componentUsage.NewService(registry)
	if semanticRuntime.Answerer != nil {
		semanticRuntime.Answerer.SetToolRunner(grpcAma.NewAppUsageToolRunner(usageService))
	}

	transitionQueueReporter := reporter.NewQueueReporter(registry)
	go func() {
		if err := transitionQueueReporter.Run(ctx, monitorHandle.Transitions); err != nil {
			logger.ErrorContext(ctx, "transition queue reporter stopped", "error", err)
		}
	}()

	return serveGRPC(ctx, registry, logger)
}

func buildQueues(ctx context.Context, registry *services.Services[any, any]) error {
	queueDSN := envs.DatabaseDSN.Value()
	if envs.DatabaseEngine.Value() == db.EngineSqlite {
		queueDSN = db.SqliteDataSourceName(queueDSN)
	}

	transitionEventQueue, err := queue.New[reporter.TransitionEvent](ctx, queue.QueueOptions{
		ConnectionString: queueDSN,
		QueueName:        workers.TransitionEventQueueName.String(),
	})
	if err != nil {
		return fmt.Errorf("create transition event queue: %w", err)
	}
	reporter.RegisterTransitionEventQueue(registry, transitionEventQueue)

	transitionEventReportedQueue, err := queue.New[workers.TransitionEventReported](ctx, queue.QueueOptions{
		ConnectionString: queueDSN,
		QueueName:        workers.TransitionEventReportedQueueName.String(),
	})
	if err != nil {
		return fmt.Errorf("create transition event reported queue: %w", err)
	}
	workers.RegisterQueues(registry, workers.Queues{
		TransitionEventReportedQueue: transitionEventReportedQueue,
	})
	return nil
}

func buildSemanticRuntime(ctx context.Context, registry *services.Services[any, any], database *db.Database, logger *slog.Logger) *semantic.Runtime {
	semanticRuntime, err := newSemanticRuntime(ctx, database, logger)
	if err != nil {
		logger.WarnContext(ctx, "semantic services unavailable", "error", err)
		unavailableReason := semanticStartupMessage(err)
		if semanticRuntime == nil {
			semanticRuntime = &semantic.Runtime{UnavailableReason: unavailableReason}
		} else {
			semanticRuntime.UnavailableReason = unavailableReason
		}
	}
	if semanticRuntime == nil {
		semanticRuntime = &semantic.Runtime{}
	}
	semantic.RegisterRuntime(registry, semanticRuntime)

	if semanticRuntime.Answerer != nil && semanticRuntime.Backfilling != nil {
		logger.InfoContext(ctx, "semantic indexing enabled",
			"embedding_model", semanticRuntime.EmbeddingModel,
			"embedding_dimension", semantic.StoreEmbeddingDimension,
		)
		logger.InfoContext(ctx, "RAG answering enabled", "rag_model", semanticRuntime.RAGModel)
		semanticRuntime.Backfilling.Start(startupSemanticBackfillLimit)
	} else if semanticRuntime.IndexStatus != nil {
		logger.WarnContext(ctx, "semantic index status enabled but answering disabled", "reason", semanticRuntime.UnavailableReason)
	} else {
		logger.WarnContext(ctx, "semantic indexing disabled until AI settings are configured")
	}

	return semanticRuntime
}

func startWorkers(ctx context.Context, registry *services.Services[any, any], runtime *workers.Runtime, semanticRuntime *semantic.Runtime) []func() {
	var cleanups []func()
	if semanticRuntime.Indexer != nil {
		queues, _ := workers.QueuesFromServices(registry)
		if queues.TransitionEventReportedQueue != nil {
			transitionEventReportedCleanup := runtime.TransitionEventIndexerWorker(ctx, queues.TransitionEventReportedQueue)
			cleanups = append(cleanups, transitionEventReportedCleanup)
		}
	}

	transitionEventQueue, _ := reporter.TransitionEventQueueFromServices(registry)
	if queue, ok := transitionEventQueue.(*queue.Queue[reporter.TransitionEvent]); ok {
		transitionEventCleanup := runtime.TransitionEventReporterWorker(ctx, queue)
		cleanups = append(cleanups, transitionEventCleanup)
	}
	return cleanups
}

func serveGRPC(ctx context.Context, registry *services.Services[any, any], logger *slog.Logger) error {
	rpcLogger := logger.With("service", "gRPC/server", "component", component)
	logTraceID := func(ctx context.Context) grpcLogging.Fields {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return grpcLogging.Fields{"traceID", span.TraceID().String()}
		}
		return nil
	}

	listenAddress := envs.GrpcListenAddress.Value().String()
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen on address %s: %w", listenAddress, err)
	}

	grpcPanicRecoveryHandler := func(ctx context.Context, p any) (err error) {
		rpcLogger.ErrorContext(ctx, "recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcLogging.UnaryServerInterceptor(interceptorLogger(rpcLogger), grpcLogging.WithFieldsFromContext(logTraceID)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			grpcLogging.StreamServerInterceptor(interceptorLogger(rpcLogger), grpcLogging.WithFieldsFromContext(logTraceID)),
		),
	)
	reflection.Register(server)
	transitionsv1.RegisterTransitionsServiceServer(server, grpcTransitions.NewServer(registry))
	amav1.RegisterAmaServiceServer(server, grpcAma.NewServer(registry))
	settingsv1.RegisterSettingsServiceServer(server, grpcSettings.NewServer(registry))
	usagev1.RegisterUsageServiceServer(server, grpcUsage.NewServer(registry))

	logger.InfoContext(ctx, "sidecar gRPC server listening", "address", listenAddress)
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}
	return nil
}

func interceptorLogger(l *slog.Logger) grpcLogging.Logger {
	return grpcLogging.LoggerFunc(func(ctx context.Context, lvl grpcLogging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func newSemanticRuntime(ctx context.Context, database *db.Database, logger *slog.Logger) (*semantic.Runtime, error) {
	settings, err := semantic.LoadAISettings(ctx, database.ReadQuerier)
	if err != nil {
		return nil, err
	}

	embeddingSlug, err := settings.EmbeddingSlug()
	if err != nil {
		return nil, err
	}
	semanticSlug, err := settings.SemanticSlug()
	if err != nil {
		return nil, err
	}
	embeddingModel := semantic.ProviderModelKey(settings.Provider, embeddingSlug)
	unavailableIndexStatus := semantic.NewIndexStatusService(database.ReadQuerier, nil, embeddingModel)

	embedder, generator, err := buildAIClients(settings, embeddingSlug, semanticSlug)
	if err != nil {
		return &semantic.Runtime{
			IndexStatus:    unavailableIndexStatus,
			EmbeddingModel: embeddingModel,
		}, err
	}
	if err := semantic.CheckEmbedderHealth(ctx, embedder); err != nil {
		return &semantic.Runtime{
			IndexStatus:    unavailableIndexStatus,
			EmbeddingModel: embedder.Model(),
			RAGModel:       generator.Model(),
		}, err
	}

	indexer := semantic.NewIndexer(database.WriteConn, database.ReadQuerier, embedder)
	backfiller := semantic.NewBackfiller(database.ReadQuerier, indexer, embedder.Model())
	backfillCoordinator := semantic.NewBackfillCoordinator(ctx, backfiller, logger.With("service", "semantic_backfill"))
	indexStatus := semantic.NewIndexStatusService(database.ReadQuerier, backfillCoordinator, embedder.Model())
	searcher := semantic.NewSearcher(database.ReadConn, embedder)

	return &semantic.Runtime{
		Answerer:       semantic.NewAnswerer(searcher, generator),
		Backfilling:    backfillCoordinator,
		IndexStatus:    indexStatus,
		Indexer:        indexer,
		EmbeddingModel: embedder.Model(),
		RAGModel:       generator.Model(),
	}, nil
}

func buildAIClients(settings semantic.AISettings, embeddingSlug string, semanticSlug string) (semantic.Embedder, semantic.Generator, error) {
	switch settings.Provider {
	case semantic.ProviderOpenRouter:
		apiKey, ok := envs.OpenRouterAPIKey.Value()
		apiKey = strings.TrimSpace(apiKey)
		if !ok || apiKey == "" {
			return nil, nil, fmt.Errorf("OpenRouter API key is not configured")
		}
		embedder, err := semantic.NewOpenRouterEmbedder(semantic.OpenRouterConfig{
			APIKey:    apiKey,
			BaseURL:   settings.OpenRouterBaseURL,
			Model:     embeddingSlug,
			Dimension: semantic.StoreEmbeddingDimension,
		})
		if err != nil {
			return nil, nil, err
		}
		generator, err := semantic.NewOpenRouterGenerator(semantic.OpenRouterConfig{
			APIKey:  apiKey,
			BaseURL: settings.OpenRouterBaseURL,
			Model:   semanticSlug,
		})
		if err != nil {
			return nil, nil, err
		}
		return embedder, generator, nil

	case semantic.ProviderOllama:
		apiKey, _ := envs.OllamaAPIKey.Value()
		apiKey = strings.TrimSpace(apiKey)
		embedder, err := semantic.NewOllamaEmbedder(semantic.OllamaConfig{
			APIKey:    apiKey,
			BaseURL:   settings.OllamaBaseURL,
			Model:     embeddingSlug,
			Dimension: semantic.StoreEmbeddingDimension,
		})
		if err != nil {
			return nil, nil, err
		}
		generator, err := semantic.NewOllamaGenerator(semantic.OllamaConfig{
			APIKey:  apiKey,
			BaseURL: settings.OllamaBaseURL,
			Model:   semanticSlug,
		})
		if err != nil {
			return nil, nil, err
		}
		return embedder, generator, nil

	default:
		return nil, nil, fmt.Errorf("unsupported AI provider %q", settings.Provider)
	}
}

func semanticStartupMessage(err error) string {
	if message, ok := semantic.AIRequestUserMessage(err); ok {
		return message
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Semantic index is unavailable."
	}
	return message
}
