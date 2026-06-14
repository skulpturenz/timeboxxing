package workers

import (
	"fmt"

	"github.com/goptics/varmq"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

func (s WorkerServices) AddTransitionEventWorker(q queue.Queue) (varmq.PersistentQueue[any], func()) {
	logger := s.logger.With("worker", "transition_event")
	queue, _, cleanup := q.NewWorker(func(j varmq.Job[any]) {
		logger.Info("received transition event job", "data", fmt.Sprintf("%+v", j.Data()))
	}, 0)

	return queue, cleanup
}
