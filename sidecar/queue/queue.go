package queue

import (
	"github.com/goptics/sqliteq"
	"github.com/goptics/varmq"
)

type Queue struct {
	queue *sqliteq.Queue
}

type QueueOptions struct {
	ConnectionString string
}

func (opts *QueueOptions) New() (*Queue, error) {
	q := Queue{}

	db := sqliteq.New(opts.ConnectionString)

	queue, err := db.NewQueue("test")
	if err != nil {
		return nil, err
	}

	q.queue = queue

	return &q, nil
}

func (q Queue) NewWorker(wf func(j varmq.Job[any]), config ...any) (varmq.PersistentQueue[any], varmq.IWorkerBinder[any], func()) {
	w := varmq.NewWorker(wf, config...)
	queue := w.WithPersistentQueue(q.queue)

	cleanup := func() {
		w.WaitUntilIdle()
	}

	return queue, w, cleanup
}
