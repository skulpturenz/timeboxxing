package queue

import (
	"context"

	"github.com/goptics/sqliteq"
	"github.com/goptics/varmq"
)

type Queue struct {
	queue *sqliteq.Queue
}

type QueueOptions struct {
	ConnectionString string
	QueueName        string
}

func (opts *QueueOptions) New(ctx context.Context) (*Queue, error) {
	q := Queue{}

	db := sqliteq.New(opts.ConnectionString)
	queueName := opts.QueueName
	if queueName == "" {
		queueName = "test"
	}

	queue, err := db.NewQueue(queueName)
	if err != nil {
		return nil, err
	}

	q.queue = queue

	return &q, nil
}

func (q Queue) NewWorker(_ context.Context, wf func(j varmq.Job[any]), config ...any) (varmq.PersistentQueue[any], varmq.IWorkerBinder[any], func()) {
	w := varmq.NewWorker(wf, config...)
	queue := w.WithPersistentQueue(q.queue)

	cleanup := func() {
		w.WaitUntilIdle()
	}

	return queue, w, cleanup
}
