package main

//go:generate go tool buf generate

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	hellov1 "github.com/skulpturenz/timeboxxing/sidecar/gen/hello/v1"
	helloserviceone "github.com/skulpturenz/timeboxxing/sidecar/grpc/hello_service_one"
	helloservicetwo "github.com/skulpturenz/timeboxxing/sidecar/grpc/hello_service_two"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
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

// interceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	dsn := envs.DatabaseDSN.Value()
	database, err := db.New(ctx, db.Options{
		Engine:         envs.DatabaseEngine.Value(),
		DataSourceName: dsn,
	})
	if err != nil {
		log.Fatalf("create database: %v", err)
	}
	defer database.Close()

	// queue
	persistedQueue := queue.QueueOptions{
		ConnectionString: dsn,
	}
	queue, err := persistedQueue.New()
	if err != nil {
		panic(err)
	}
	transitionEventQueue, transitionEventQueueCleanup := workers.AddTransitionEventWorker(*queue)
	defer transitionEventQueueCleanup()
	transitions, err := monitor.Start(ctx, logger.With("service", "monitor"), monitor.Config{})
	if err != nil {
		log.Fatalf("start monitor: %v", err)
	}
	databaseReporter := reporter.NewDatabaseReporter(database.Conn)
	go func() {
		for transition := range transitions {
			if ok := transitionEventQueue.Add(transition); !ok {
				logger.Warn("failed to add transition event to transition event queue", "reason", transition.Reason)
			}

			if err := databaseReporter.Record(ctx, transition); err != nil {
				logger.Error("failed to report transition event", "reason", transition.Reason, "error", err)
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
		log.Fatalf("listen on %s: %v", listenAddress, err)
	}

	grpcPanicRecoveryHandler := func(p any) (err error) {
		rpcLogger.Error("recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
		),
	)
	reflection.Register(server)
	hellov1.RegisterHelloServiceOneServer(server, helloserviceone.HelloServiceOneServer{})
	hellov1.RegisterHelloServiceTwoServer(server, helloservicetwo.HelloServiceTwoServer{})

	log.Printf("sidecar gRPC server listening on %s", listenAddress)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve grpc: %v", err)
	}
}
