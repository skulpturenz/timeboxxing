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
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	hellov1 "github.com/skulpturenz/timeboxxing/sidecar/gen/hello/v1"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	helloserviceone "github.com/skulpturenz/timeboxxing/sidecar/grpc/hello_service_one"
	helloservicetwo "github.com/skulpturenz/timeboxxing/sidecar/grpc/hello_service_two"
	grpcTransitions "github.com/skulpturenz/timeboxxing/sidecar/grpc/transitions"
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
	component = "grpc-example"
)

type appServices struct {
	databaseReporter *reporter.DatabaseReporter
	answerer         *semantic.Answerer
	transitions      *componentTransitions.Service
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
	persistedQueue := queue.QueueOptions{
		ConnectionString: queueDSN,
	}
	queue, err := persistedQueue.New(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "create queue", "error", err)
		panic(err)
	}
	workerServices := workers.NewWorkerServices(workers.WorkerServicesParams{
		ReadQueries:  database.ReadQuerier,
		WriteQueries: database.WriteQuerier,
		Logger:       logger.With("service", "workers"),
	})
	transitionEventQueue, transitionEventQueueCleanup := workerServices.AddTransitionEventWorker(ctx, *queue)
	defer transitionEventQueueCleanup()
	transitions, err := monitor.Start(ctx, logger.With("service", "monitor"), monitor.Config{})
	if err != nil {
		logger.ErrorContext(ctx, "start monitor", "error", err)
		panic(err)
	}
	services := appServices{
		databaseReporter: reporter.NewDatabaseReporter(database.WriteConn),
		transitions: componentTransitions.NewService(componentTransitions.NewServiceParams{
			Querier: database.ReadQuerier,
		}),
	}
	openRouterAPIKey := strings.TrimSpace(envs.OpenRouterAPIKey.Value())
	if openRouterAPIKey == "" {
		logger.InfoContext(ctx, "semantic indexing and RAG answering disabled; SIDECAR_OPENROUTER_API_KEY is not configured")
	} else {
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
		generator, err := semantic.NewOpenRouterGenerator(semantic.OpenRouterConfig{
			APIKey:  openRouterAPIKey,
			BaseURL: envs.OpenRouterBaseURL.Value(),
			Model:   envs.RAGModel.Value(),
		})
		if err != nil {
			logger.ErrorContext(ctx, "create semantic generator", "error", err)
			panic(err)
		}

		searcher := semantic.NewSearcher(database.ReadConn, embedder)
		services.databaseReporter = reporter.NewDatabaseReporterWithIndexer(database.WriteConn, semantic.NewIndexer(database.WriteConn, database.ReadQuerier, embedder))
		services.answerer = semantic.NewAnswerer(searcher, generator)
		logger.InfoContext(ctx, "semantic indexing enabled", "embedding_model", embedder.Model(), "embedding_dimension", embedder.Dimension())
		logger.InfoContext(ctx, "RAG answering enabled", "rag_model", generator.Model())
	}
	go func() {
		for transition := range transitions {
			if ok := transitionEventQueue.Add(transition); !ok {
				logger.WarnContext(ctx, "failed to add transition event to transition event queue", "reason", transition.Reason)
			}

			eventID, err := services.databaseReporter.Record(ctx, transition)
			if err != nil {
				logger.ErrorContext(ctx, "failed to report transition event", "reason", transition.Reason, "error", err)
				continue
			}
			if eventID != 0 {
				if err := services.transitions.PublishTransitionEvent(ctx, componentTransitions.PublishTransitionEventParams{ID: eventID}); err != nil {
					logger.ErrorContext(ctx, "failed to publish transition event", "transition_event_id", eventID, "error", err)
				}
			}
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
	hellov1.RegisterHelloServiceOneServer(server, helloserviceone.HelloServiceOneServer{})
	hellov1.RegisterHelloServiceTwoServer(server, helloservicetwo.HelloServiceTwoServer{})
	transitionsv1.RegisterTransitionsServiceServer(server, grpcTransitions.NewServer(grpcTransitions.NewServerParams{
		Transitions: services.transitions,
	}))

	logger.InfoContext(ctx, "sidecar gRPC server listening", "address", listenAddress)
	if err := server.Serve(listener); err != nil {
		logger.ErrorContext(ctx, "serve grpc", "error", err)
		panic(err)
	}
}
