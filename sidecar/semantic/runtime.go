package semantic

import "github.com/skulpturenz/timeboxxing/sidecar/services"

type Runtime struct {
	Answerer          *Answerer
	Backfilling       *BackfillCoordinator
	IndexStatus       *IndexStatusService
	Indexer           *Indexer
	EmbeddingModel    string
	RAGModel          string
	UnavailableReason string
}

type runtimeKey struct{}

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
