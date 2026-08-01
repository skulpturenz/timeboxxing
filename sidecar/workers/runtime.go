package workers

import (
	"context"
	"log/slog"

	"github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// transitionEventIndexer indexes a finalized transition event for semantic search.
// *semantic.Indexer satisfies it.
type transitionEventIndexer interface {
	IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error)
}

type Runtime struct {
	logger  *slog.Logger
	queues  Queues
	indexer transitionEventIndexer
}

type Queues struct {
	// TransitionEventReportedIn is the producer side — send a finalized transition event to enqueue
	// it durably. TransitionEventReportedOut is the consumer side — the queue delivers each persisted
	// event here for the indexer worker to process.
	TransitionEventReportedIn  chan<- TransitionEventReported
	TransitionEventReportedOut <-chan TransitionEventReported
}

type queuesKey struct{}
type runtimeKey struct{}

func RegisterQueues(registry *services.Services[any, any], queues Queues) {
	services.Set(registry, queuesKey{}, queues)
}

func QueuesFromServices(registry *services.Services[any, any]) (Queues, bool) {
	service, ok := services.Get[Queues](registry, queuesKey{})
	if !ok {
		return Queues{}, false
	}
	return service.Unwrap(), true
}

func RegisterRuntime(registry *services.Services[any, any], runtime *Runtime) {
	services.Set(registry, runtimeKey{}, runtime)
}

func RuntimeFromServices(registry *services.Services[any, any]) (*Runtime, bool) {
	service, ok := services.Get[*Runtime](registry, runtimeKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func NewRuntime(registry *services.Services[any, any]) *Runtime {
	queues, _ := QueuesFromServices(registry)
	var indexer transitionEventIndexer
	if semanticRuntime, ok := semantic.RuntimeFromServices(registry); ok && semanticRuntime != nil && semanticRuntime.Indexer != nil {
		indexer = semanticRuntime.Indexer
	}

	runtime := &Runtime{
		logger:  logging.LoggerFromServices(registry).With("service", "workers"),
		queues:  queues,
		indexer: indexer,
	}
	RegisterRuntime(registry, runtime)
	return runtime
}
