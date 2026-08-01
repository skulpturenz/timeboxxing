package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"strings"
	"time"

	"github.com/goptics/sqliteq"
	grpcLogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelineConverters "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/converters"
	timelineModels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	grpcAma "github.com/skulpturenz/timeboxxing/sidecar/grpc/ama"
	grpcProjects "github.com/skulpturenz/timeboxxing/sidecar/grpc/projects"
	grpcSettings "github.com/skulpturenz/timeboxxing/sidecar/grpc/settings"
	grpcTimesheets "github.com/skulpturenz/timeboxxing/sidecar/grpc/timesheets"
	grpcUsage "github.com/skulpturenz/timeboxxing/sidecar/grpc/usage"
	sidecarLogging "github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/stack"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/observability"
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

	var sqliteVectorExtensionPath *string
	if path, ok := envs.SQLiteVectorExtensionPath.Value(); ok {
		sqliteVectorExtensionPath = &path
	}
	databaseKey, _ := envs.ResolvedDatabaseKey()
	dsn := db.NewDSN(envs.DB_DSN.Value())
	dsn.SetJournalMode(enumsjournalmode.WAL)
	dsn.EnableFK()
	dsn.SetBusyTimeout(5 * time.Second)
	if databaseKey != "" {
		if err := dsn.EnableEncryption(databaseKey); err != nil {
			return fmt.Errorf("configure database encryption: %w", err)
		}
	}
	database, err := db.New(ctx, db.Options{
		DSN:                       dsn,
		SQLiteVectorExtensionPath: sqliteVectorExtensionPath,
	})
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer database.Close()
	db.Register(registry, database)

	// Cancelable so shutdown can be triggered explicitly (e.g. if serveGRPC returns while the
	// parent ctx is still live). Downstream — the queue worker and indexer consumer — derive from
	// this ctx.
	ctx, cancel := context.WithCancel(ctx)

	closeQueues, err := buildQueues(ctx, registry)
	if err != nil {
		cancel()
		return err
	}
	defer func() {
		if err := closeQueues(); err != nil {
			logger.ErrorContext(ctx, "close queue manager", "error", err)
		}
	}()

	semanticRuntime := buildSemanticRuntime(ctx, registry, database, logger)
	workerRuntime := workers.NewRuntime(registry)
	for _, cleanup := range startWorkers(ctx, registry, workerRuntime, semanticRuntime) {
		defer cleanup()
	}
	// Registered last so it runs first on return: cancel stops the queue worker and the indexer
	// consumer, which lets their drain cleanups above unblock and the queue manager close last.
	defer cancel()

	startStartupSemanticBackfill(semanticRuntime)

	if semanticRuntime.Answerer != nil {
		semanticRuntime.Answerer.SetToolRunner(grpcAma.NewAppUsageToolRunner(registry))
	}

	// The removed Service.Project ingest carried the semantic-indexing enqueuer via
	// componentTimeline.Options so finalized timeline entries were pushed onto
	// TransitionEventReportedQueue. The Command ingest does not enqueue (indexing is intended to move
	// to the GetUnindexedForegroundProcesses stream). Preserved commented-out so the enqueue wiring is
	// trivial to re-enable when the stream/read features are wired.
	// timelineOptions := componentTimeline.Options{}
	// if semanticRuntime.Indexer != nil {
	// 	if queues, ok := workers.QueuesFromServices(registry); ok && queues.TransitionEventReportedQueue != nil {
	// 		timelineOptions.Enqueuer = workers.NewTransitionEventReportedEnqueuer(queues.TransitionEventReportedQueue)
	// 	}
	// }
	startForegroundProjection(ctx, registry, logger.With("service", "timeline"))

	return serveGRPC(ctx, registry, logger)
}

// startForegroundProjection wires the foreground monitor to the timeline ingest: it polls the OS
// foreground process, dedups to change events via the pub/sub reporter, enriches each one (app
// metadata, browser tab/URL, location), adapts it to the timeline model, and feeds it to the
// CommandSubscribeReporter which persists each observation into the timeline event store.
func startForegroundProjection(ctx context.Context, registry *services.Services[any, any], logger *slog.Logger) {
	enrich, permissions := stack.Stack()
	m, err := monitor.New(ctx, monitor.Options{Permissions: permissions})
	if err != nil {
		logger.WarnContext(ctx, "foreground monitor unavailable", "error", err)
		return
	}

	pubsub, _ := reporter.From(ctx, m.Stream)
	events := pubsub.Subscribe("timeline")

	// Bridge the reporter's monitor.ForegroundProcess stream into the models.ForegroundProcess channel
	// the ingest command consumes, enriching and adapting each observation en route.
	var converter timelineConverters.MonitorForegroundProcessConverter
	modelStream := make(chan timelineModels.ForegroundProcess)
	go func() {
		defer close(modelStream)
		for foregroundProcess := range events {
			enriched, _ := enrich(ctx, foregroundProcess)
			select {
			case <-ctx.Done():
				return
			case modelStream <- converter.ToForegroundProcess(enriched):
			}
		}
	}()

	subscribeReporter := &componentTimeline.CommandSubscribeReporter{Chan: modelStream}
	go func() {
		if err := subscribeReporter.Exec(ctx, registry); err != nil {
			logger.ErrorContext(ctx, "timeline ingest stopped", "error", err)
		}
	}()
}

// buildQueues opens the durable transition-event queue and registers its producer/consumer channel
// ends. It returns a cleanup that closes the SQLite manager it owns; the queue's own worker is torn
// down when ctx is cancelled (which closes the delivered outChan).
func buildQueues(ctx context.Context, registry *services.Services[any, any]) (func() error, error) {
	// Reuse the exact keyed DSN the writer/reader use so the queue connection is encrypted
	// identically — sqliteq opens the same file via a hardcoded sql.Open("sqlite3", ...).
	// buildQueues runs immediately after db.Register, so the database is always present.
	database, _ := db.FromServices(registry)
	queueDSN := database.DSN.String()

	manager := sqliteq.New(queueDSN)

	inChan := make(chan workers.TransitionEventReported)
	outChan, err := queue.New[workers.TransitionEventReported](ctx, queue.QueueOptions[workers.TransitionEventReported]{
		Manager: manager,
		Name:    workers.TransitionEventReportedQueueName.String(),
		InChan:  inChan,
	})
	if err != nil {
		_ = manager.Close()
		return nil, fmt.Errorf("create transition event reported queue: %w", err)
	}

	workers.RegisterQueues(registry, workers.Queues{
		TransitionEventReportedIn:  inChan,
		TransitionEventReportedOut: outChan,
	})

	return manager.Close, nil
}

func buildSemanticRuntime(ctx context.Context, registry *services.Services[any, any], database *db.Database, logger *slog.Logger) *semantic.Runtime {
	semanticRuntime, err := newSemanticRuntime(ctx, database, logger, transitionEventBackfillEnqueuerFromServices(registry))
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
	} else if semanticRuntime.IndexStatus != nil {
		logger.WarnContext(ctx, "semantic index status enabled but answering disabled", "reason", semanticRuntime.UnavailableReason)
	} else {
		logger.WarnContext(ctx, "semantic indexing disabled until AI settings are configured")
	}

	return semanticRuntime
}

func transitionEventBackfillEnqueuerFromServices(registry *services.Services[any, any]) semantic.TransitionEventEnqueuer {
	queues, _ := workers.QueuesFromServices(registry)
	if queues.TransitionEventReportedIn == nil {
		return nil
	}
	return workers.NewTransitionEventReportedEnqueuer(queues.TransitionEventReportedIn)
}

func startStartupSemanticBackfill(runtime *semantic.Runtime) {
	if runtime != nil && runtime.Answerer != nil && runtime.Backfilling != nil {
		runtime.Backfilling.Start(startupSemanticBackfillLimit)
	}
}

func startWorkers(ctx context.Context, registry *services.Services[any, any], runtime *workers.Runtime, semanticRuntime *semantic.Runtime) []func() {
	var cleanups []func()
	if semanticRuntime.Indexer != nil {
		queues, _ := workers.QueuesFromServices(registry)
		if queues.TransitionEventReportedOut != nil {
			transitionEventReportedCleanup := runtime.TransitionEventIndexerWorker(ctx, queues.TransitionEventReportedOut)
			cleanups = append(cleanups, transitionEventReportedCleanup)
		}
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

	listenAddress := envs.GRPC_LISTEN_ADDRESS.Value().String()
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
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(grpcPanicRecoveryHandler)),
			observability.UnaryServerInterceptor(),
			grpcLogging.UnaryServerInterceptor(interceptorLogger(rpcLogger), grpcLogging.WithFieldsFromContext(logTraceID)),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandlerContext(grpcPanicRecoveryHandler)),
			observability.StreamServerInterceptor(),
			grpcLogging.StreamServerInterceptor(interceptorLogger(rpcLogger), grpcLogging.WithFieldsFromContext(logTraceID)),
		),
	)
	reflection.Register(server)
	amav1.RegisterAmaServiceServer(server, grpcAma.NewServer(registry))
	projectsv1.RegisterProjectsServiceServer(server, grpcProjects.NewServer(registry))
	settingsv1.RegisterSettingsServiceServer(server, grpcSettings.NewServer(registry))
	timesheetsv1.RegisterTimesheetsServiceServer(server, grpcTimesheets.NewServer(registry))
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

func newSemanticRuntime(ctx context.Context, database *db.Database, logger *slog.Logger, backfillEnqueuer semantic.TransitionEventEnqueuer) (*semantic.Runtime, error) {
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
	unavailableIndexStatus := semantic.NewIndexStatusService(database.ReadQuerier, nil, settings.EmbeddingModelID)

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

	vectorStore := semantic.NewSQLiteVectorStore(database.ReadConn)
	if err := vectorStore.Check(ctx); err != nil {
		return &semantic.Runtime{
			IndexStatus:    unavailableIndexStatus,
			EmbeddingModel: embedder.Model(),
			RAGModel:       generator.Model(),
		}, err
	}

	indexer := semantic.NewIndexer(database.WriteQuerier, database.ReadQuerier, embedder, settings.EmbeddingModelID)
	backfiller := semantic.NewBackfiller(database.ReadQuerier, backfillEnqueuer, settings.EmbeddingModelID)
	backfillCoordinator := semantic.NewBackfillCoordinator(ctx, backfiller, logger.With("service", "semantic_backfill"))
	indexStatus := semantic.NewIndexStatusService(database.ReadQuerier, backfillCoordinator, settings.EmbeddingModelID)
	searcher := semantic.NewSearcher(database.ReadConn, embedder, settings.EmbeddingModelID, vectorStore)

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
		apiKey, ok := envs.ResolvedOpenRouterAPIKey()
		if !ok || apiKey == "" {
			return nil, nil, fmt.Errorf("OpenRouter API key is not configured")
		}
		embedder, err := semantic.NewOpenRouterEmbedder(semantic.OpenRouterConfig{
			APIKey:    apiKey,
			BaseURL:   settings.BaseURL(),
			Model:     embeddingSlug,
			Dimension: semantic.StoreEmbeddingDimension,
		})
		if err != nil {
			return nil, nil, err
		}
		generator, err := semantic.NewOpenRouterGenerator(semantic.OpenRouterConfig{
			APIKey:  apiKey,
			BaseURL: settings.BaseURL(),
			Model:   semanticSlug,
		})
		if err != nil {
			return nil, nil, err
		}
		return embedder, generator, nil

	case semantic.ProviderOllama:
		apiKey, _ := envs.ResolvedOllamaAPIKey()
		embedder, err := semantic.NewOllamaEmbedder(semantic.OllamaConfig{
			APIKey:    apiKey,
			BaseURL:   settings.BaseURL(),
			Model:     embeddingSlug,
			Dimension: semantic.StoreEmbeddingDimension,
		})
		if err != nil {
			return nil, nil, err
		}
		generator, err := semantic.NewOllamaGenerator(semantic.OllamaConfig{
			APIKey:  apiKey,
			BaseURL: settings.BaseURL(),
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
