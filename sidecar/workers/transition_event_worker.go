package workers

import (
	"context"
	"fmt"

	"github.com/goptics/varmq"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

func (s WorkerServices) AddTransitionEventWorker(ctx context.Context, q queue.Queue) (varmq.PersistentQueue[any], func()) {
	logger := s.logger.With("worker", "transition_event")
	queue, _, cleanup := q.NewWorker(ctx, func(j varmq.Job[any]) {
		logger.InfoContext(ctx, "received transition event job", "data", fmt.Sprintf("%+v", j.Data()))
	}, 0)

	return queue, cleanup
}
