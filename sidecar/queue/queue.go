package queue

import (
	"context"
	"fmt"

	"github.com/goptics/sqliteq"
	"github.com/goptics/varmq"
)

type Queue[T any] struct {
	backend *sqliteq.Queue
	queue   varmq.PersistentQueue[T]
}

type QueueOptions struct {
	ConnectionString string
	QueueName        string
}

func New[T any](ctx context.Context, opts QueueOptions) (*Queue[T], error) {
	db := sqliteq.New(opts.ConnectionString)
	queueName := opts.QueueName
	if queueName == "" {
		queueName = "test"
	}

	queue, err := db.NewQueue(queueName)
	if err != nil {
		return nil, err
	}

	return &Queue[T]{backend: queue}, nil
}

func (q *Queue[T]) Add(item T) error {
	if q.queue == nil {
		return fmt.Errorf("queue has no worker")
	}

	if ok := q.queue.Add(item); !ok {
		return fmt.Errorf("add item to queue")
	}

	return nil
}

func (q *Queue[T]) AddWorker(_ context.Context, wf func(j varmq.Job[T]), config ...any) func() {
	w := varmq.NewWorker(wf, config...)
	q.queue = w.WithPersistentQueue(q.backend)

	cleanup := func() {
		w.WaitUntilIdle()
	}

	return cleanup
}
