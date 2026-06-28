package main

//go:generate go tool buf generate

import (
	"context"
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
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	grpcAma "github.com/skulpturenz/timeboxxing/sidecar/grpc/ama"
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
	transitions *componentTransitions.Service
	usage       *componentUsage.Service
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

	openRouterAPIKey := strings.TrimSpace(envs.OpenRouterAPIKey.Value())
	embedder, err := semantic.NewOpenRouterEmbedder(semantic.OpenRouterConfig{
		APIKey:    openRouterAPIKey,
		BaseURL:   envs.OpenRouterBaseURL.Value(),
		Model:     envs.EmbeddingModel.Value(),
		Dimension: int(envs.EmbeddingDimension.Value()),
	})
	if err != nil {
		logger.ErrorContext(ctx, "create semantic embedder", "error", err)
		panic(err)
	}
	if err := semantic.CheckEmbedderHealth(ctx, embedder); err != nil {
		logger.ErrorContext(ctx, "check semantic embedder", "error", err)
		panic(err)
	}
	generator, err := semantic.NewOpenRouterGenerator(semantic.OpenRouterConfig{
		APIKey:  openRouterAPIKey,
		BaseURL: envs.OpenRouterBaseURL.Value(),
		Model:   envs.RAGModel.Value(),
	})
	if err != nil {
		logger.ErrorContext(ctx, "create semantic generator", "error", err)
		panic(err)
	}
	transitionEventIndexer := semantic.NewIndexer(database.WriteConn, database.ReadQuerier, embedder)
	semanticBackfiller := semantic.NewBackfiller(database.ReadQuerier, transitionEventIndexer)
	semanticBackfillCoordinator := semantic.NewBackfillCoordinator(ctx, semanticBackfiller, logger.With("service", "semantic_backfill"))
	semanticIndexStatus := semantic.NewIndexStatusService(database.ReadQuerier, semanticBackfillCoordinator)
	searcher := semantic.NewSearcher(database.ReadConn, embedder)

	transitionsService := componentTransitions.NewService(componentTransitions.NewServiceParams{
		Querier: database.ReadQuerier,
	})
	services := appServices{
		answerer:    semantic.NewAnswerer(searcher, generator),
		backfilling: semanticBackfillCoordinator,
		indexStatus: semanticIndexStatus,
		transitions: transitionsService,
	}
	logger.InfoContext(ctx, "semantic indexing enabled", "embedding_model", embedder.Model(), "embedding_dimension", embedder.Dimension())
	logger.InfoContext(ctx, "RAG answering enabled", "rag_model", generator.Model())

	semanticBackfillCoordinator.Start(startupSemanticBackfillLimit)

	workerServices := workers.WorkerServices{
		ReadQueries:            database.ReadQuerier,
		WriteQueries:           database.WriteQuerier,
		ReadConn:               database.ReadConn,
		WriteConn:              database.WriteConn,
		Logger:                 logger.With("service", "workers"),
		TransitionEventIndexer: transitionEventIndexer,
		Transitions:            services.transitions,
		Queues: workers.WorkerQueues{
			TransitionEventReportedQueue: transitionEventReportedQueue,
			TransitionEventQueue:         transitionEventQueue,
		},
	}

	// workers
	transitionEventReportedCleanup := workerServices.TransitionEventIndexerWorker(ctx, transitionEventReportedQueue)
	defer transitionEventReportedCleanup()

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
		Answerer:    services.answerer,
		Backfilling: services.backfilling,
		IndexStatus: services.indexStatus,
		Logger:      logger.With("service", "gRPC/ama"),
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
