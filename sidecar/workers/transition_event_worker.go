package workers

import (
	"github.com/goptics/varmq"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

func AddTransitionEventWorker(q queue.Queue) (varmq.PersistentQueue[any], func()) {
	queue, _, cleanup := q.NewWorker(func(j varmq.Job[any]) {

	}, 0)

	return queue, cleanup
}
