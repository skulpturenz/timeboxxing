package workers

import (
	"log/slog"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Runtime struct {
	logger      *slog.Logger
	queues      Queues
	indexer     reporter.TransitionEventIndexer
	transitions *componentTransitions.Service
}

type Queues struct {
	TransitionEventReportedQueue *queue.Queue[TransitionEventReported]
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
	transitions, _ := componentTransitions.ServiceFromServices(registry)
	var indexer reporter.TransitionEventIndexer
	if semanticRuntime, ok := semantic.RuntimeFromServices(registry); ok && semanticRuntime != nil {
		indexer = semanticRuntime.Indexer
	}

	runtime := &Runtime{
		logger:      logging.LoggerFromServices(registry).With("service", "workers"),
		queues:      queues,
		indexer:     indexer,
		transitions: transitions,
	}
	RegisterRuntime(registry, runtime)
	return runtime
}
