package workers

import (
	"log/slog"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type WorkerServices struct {
	readQueries  queries.Querier
	writeQueries queries.Querier
	logger       *slog.Logger
}

type WorkerServicesParams struct {
	ReadQueries  queries.Querier
	WriteQueries queries.Querier
	Logger       *slog.Logger
}

func NewWorkerServices(params WorkerServicesParams) WorkerServices {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return WorkerServices{
		readQueries:  params.ReadQueries,
		writeQueries: params.WriteQueries,
		logger:       logger,
	}
}
