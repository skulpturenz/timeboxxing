package ama

import (
	"context"
	"log/slog"

	"github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	amav1.UnimplementedAmaServiceServer

	answerer          *semantic.Answerer
	backfilling       BackfillCoordinator
	indexStatus       IndexStatusProvider
	unavailableReason string
	logger            *slog.Logger
}

type BackfillCoordinator interface {
	HasMissing(ctx context.Context) (bool, error)
	Start(limit int64) bool
}

type IndexStatusProvider interface {
	Status(ctx context.Context) (semantic.IndexStatus, error)
}

type serverKey struct{}

func RegisterServer(registry *services.Services[any, any], server *Server) {
	services.Set(registry, serverKey{}, server)
}

func ServerFromServices(registry *services.Services[any, any]) (*Server, bool) {
	service, ok := services.Get[*Server](registry, serverKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func NewServer(registry *services.Services[any, any]) *Server {
	logger := logging.LoggerFromServices(registry).With("service", "gRPC/ama")
	runtime, _ := semantic.RuntimeFromServices(registry)
	server := &Server{
		logger: logger,
	}
	if runtime != nil {
		server.answerer = runtime.Answerer
		server.backfilling = runtime.Backfilling
		server.indexStatus = runtime.IndexStatus
		server.unavailableReason = runtime.UnavailableReason
	}
	RegisterServer(registry, server)
	return server
}
