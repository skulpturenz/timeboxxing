package main

//go:generate go tool buf generate

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/lmittmann/tint"
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	grpcAma "github.com/skulpturenz/timeboxxing/sidecar/grpc/ama"
	grpcSettings "github.com/skulpturenz/timeboxxing/sidecar/grpc/settings"
	grpcUsage "github.com/skulpturenz/timeboxxing/sidecar/grpc/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
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

type appServices struct {
	answerer    *semantic.Answerer
	backfilling *semantic.BackfillCoordinator
	indexStatus *semantic.IndexStatusService
	indexer     *semantic.Indexer
	transitions *componentTransitions.Service
	usage       *componentUsage.Service
}

type semanticServices struct {
	answerer       *semantic.Answerer
	backfilling    *semantic.BackfillCoordinator
	indexStatus    *semantic.IndexStatusService
	indexer        *semantic.Indexer
	embeddingModel string
	ragModel       string
}

// interceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func main() {
	ctx := context.Background()
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{}))
	slog.SetDefault(logger)

	databaseEngine := envs.DatabaseEngine.Value()
	dsn := envs.DatabaseDSN.Value()
	database, err := db.New(ctx, db.Options{
		Engine:         databaseEngine,
		DataSourceName: dsn,
	})
	if err != nil {
		logger.ErrorContext(ctx, "create database", "error", err)
		panic(err)
	}
	defer database.Close()

	// queue
	queueDSN := dsn
	if databaseEngine == db.EngineSqlite {
		queueDSN = db.SqliteDataSourceName(queueDSN)
	}
	transitionEventQueue, err := queue.New[reporter.TransitionEvent](ctx, queue.QueueOptions{
		ConnectionString: queueDSN,
		QueueName:        workers.TransitionEventQueueName.String(),
	})
	if err != nil {
		logger.ErrorContext(ctx, "create transition event queue", "error", err)
		panic(err)
	}
	transitionEventReportedQueue, err := queue.New[workers.TransitionEventReported](ctx, queue.QueueOptions{
		ConnectionString: queueDSN,
		QueueName:        workers.TransitionEventReportedQueueName.String(),
	})
	if err != nil {
		logger.ErrorContext(ctx, "create transition event reported queue", "error", err)
		panic(err)
	}

	semanticRuntime, err := newSemanticServices(ctx, database, logger)
	semanticUnavailableReason := ""
	if err != nil {
		logger.WarnContext(ctx, "semantic services unavailable", "error", err)
		semanticUnavailableReason = semanticStartupMessage(err)
	}

	transitionsService := componentTransitions.NewService(componentTransitions.NewServiceParams{
		Querier: database.ReadQuerier,
	})
	services := appServices{
		transitions: transitionsService,
	}
	if semanticRuntime != nil {
		services.answerer = semanticRuntime.answerer
		services.backfilling = semanticRuntime.backfilling
		services.indexStatus = semanticRuntime.indexStatus
		services.indexer = semanticRuntime.indexer
	}
	if semanticRuntime != nil {
		logger.InfoContext(ctx, "semantic indexing enabled",
			"embedding_model", semanticRuntime.embeddingModel,
			"embedding_dimension", semantic.StoreEmbeddingDimension,
		)
		logger.InfoContext(ctx, "RAG answering enabled", "rag_model", semanticRuntime.ragModel)
		semanticRuntime.backfilling.Start(startupSemanticBackfillLimit)
	} else {
		logger.WarnContext(ctx, "semantic indexing disabled until AI settings are configured")
	}

	workerServices := workers.WorkerServices{
		ReadQueries:            database.ReadQuerier,
		WriteQueries:           database.WriteQuerier,
		ReadConn:               database.ReadConn,
		WriteConn:              database.WriteConn,
		Logger:                 logger.With("service", "workers"),
		TransitionEventIndexer: services.indexer,
		Transitions:            services.transitions,
		Queues: workers.WorkerQueues{
			TransitionEventReportedQueue: transitionEventReportedQueue,
			TransitionEventQueue:         transitionEventQueue,
		},
	}

	// workers
	if services.indexer != nil {
		transitionEventReportedCleanup := workerServices.TransitionEventIndexerWorker(ctx, transitionEventReportedQueue)
		defer transitionEventReportedCleanup()
	} else {
		logger.WarnContext(ctx, "transition event semantic indexer worker disabled")
	}

	transitionEventCleanup := workerServices.TransitionEventReporterWorker(ctx, transitionEventQueue)
	defer transitionEventCleanup()

	monitorHandle, err := monitor.Start(ctx, logger.With("service", "monitor"), monitor.Config{})
	if err != nil {
		logger.ErrorContext(ctx, "start monitor", "error", err)
		panic(err)
	}
	services.usage = componentUsage.NewService(componentUsage.NewServiceParams{
		Transitions:    transitionsService,
		ActiveSessions: monitorHandle,
	})
	if services.answerer != nil {
		services.answerer.SetToolRunner(grpcAma.NewAppUsageToolRunner(services.usage))
	}
	transitionQueueReporter := reporter.NewQueueReporter(transitionEventQueue)
	go func() {
		if err := transitionQueueReporter.Run(ctx, monitorHandle.Transitions); err != nil {
			logger.ErrorContext(ctx, "transition queue reporter stopped", "error", err)
		}
	}()

	// grpc
	rpcLogger := logger.With("service", "gRPC/server", "component", component)
	logTraceID := func(ctx context.Context) logging.Fields {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return logging.Fields{"traceID", span.TraceID().String()}
		}
		return nil
	}

	listenAddress := envs.GrpcListenAddress.Value().String()
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		logger.ErrorContext(ctx, "listen on address", "address", listenAddress, "error", err)
		panic(err)
	}

	grpcPanicRecoveryHandler := func(ctx context.Context, p any) (err error) {
		rpcLogger.ErrorContext(ctx, "recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
		),
	)
	reflection.Register(server)
	amav1.RegisterAmaServiceServer(server, grpcAma.NewServer(grpcAma.NewServerParams{
		Answerer:          services.answerer,
		Backfilling:       services.backfilling,
		IndexStatus:       services.indexStatus,
		UnavailableReason: semanticUnavailableReason,
		Logger:            logger.With("service", "gRPC/ama"),
	}))
	settingsv1.RegisterSettingsServiceServer(server, grpcSettings.NewServer(grpcSettings.NewServerParams{
		Querier: database.WriteQuerier,
	}))
	usagev1.RegisterUsageServiceServer(server, grpcUsage.NewServer(grpcUsage.NewServerParams{
		Usage: services.usage,
	}))

	logger.InfoContext(ctx, "sidecar gRPC server listening", "address", listenAddress)
	if err := server.Serve(listener); err != nil {
		logger.ErrorContext(ctx, "serve grpc", "error", err)
		panic(err)
	}
}

func newSemanticServices(ctx context.Context, database *db.Database, logger *slog.Logger) (*semanticServices, error) {
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

	embedder, generator, err := buildAIClients(settings, embeddingSlug, semanticSlug)
	if err != nil {
		return nil, err
	}
	if err := semantic.CheckEmbedderHealth(ctx, embedder); err != nil {
		return nil, err
	}

	indexer := semantic.NewIndexer(database.WriteConn, database.ReadQuerier, embedder)
	backfiller := semantic.NewBackfiller(database.ReadQuerier, indexer, embedder.Model())
	backfillCoordinator := semantic.NewBackfillCoordinator(ctx, backfiller, logger.With("service", "semantic_backfill"))
	indexStatus := semantic.NewIndexStatusService(database.ReadQuerier, backfillCoordinator, embedder.Model())
	searcher := semantic.NewSearcher(database.ReadConn, embedder)

	return &semanticServices{
		answerer:       semantic.NewAnswerer(searcher, generator),
		backfilling:    backfillCoordinator,
		indexStatus:    indexStatus,
		indexer:        indexer,
		embeddingModel: embedder.Model(),
		ragModel:       generator.Model(),
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
