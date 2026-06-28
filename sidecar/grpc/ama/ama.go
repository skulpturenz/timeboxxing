package ama

import (
	"context"
	"log/slog"

	"github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
)

type Server struct {
	amav1.UnimplementedAmaServiceServer

	answerer    *semantic.Answerer
	backfilling BackfillCoordinator
	indexStatus IndexStatusProvider
	logger      *slog.Logger
}

type NewServerParams struct {
	Answerer    *semantic.Answerer
	Backfilling BackfillCoordinator
	IndexStatus IndexStatusProvider
	Logger      *slog.Logger
}

type BackfillCoordinator interface {
	HasMissing(ctx context.Context) (bool, error)
	Start(limit int64) bool
}

type IndexStatusProvider interface {
	Status(ctx context.Context) (semantic.IndexStatus, error)
}

func NewServer(params NewServerParams) *Server {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		answerer:    params.Answerer,
		backfilling: params.Backfilling,
		indexStatus: params.IndexStatus,
		logger:      logger,
	}
}
