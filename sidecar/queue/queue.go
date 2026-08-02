package queue

import (
	"context"

	"github.com/goptics/sqliteq"
	"github.com/goptics/varmq"
	"github.com/negrel/assert"
)

type Options[T any] struct {
	Manager sqliteq.Queues
	Name    string
	InChan  <-chan T
}

func New[T any](ctx context.Context, opts Options[T]) (<-chan T, error) {
	assert.NotZero(opts.Name)
	assert.NotNil(opts.Manager)
	assert.NotNil(opts.InChan)

	resultChan := make(chan T)

	q, err := opts.Manager.NewQueue(opts.Name)
	assert.NoError(err)
	if err != nil {
		return nil, err
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)

	w := varmq.NewWorker(func(j varmq.Job[T]) {
		select {
		case <-ctxWithCancel.Done():
			return
		case resultChan <- j.Data():
		}
	})

	pq := w.WithPersistentQueue(q)

	cleanup := func() {
		cancel()

		stopErr := w.StopAndWait()
		assert.NoError(stopErr)

		close(resultChan)
	}

	go func() {
		for {
			select {
			case <-ctxWithCancel.Done():
				cleanup()
				return
			case v, ok := <-opts.InChan:
				if !ok {
					cleanup()
					return
				}

				ok = pq.Add(v)
				assert.True(ok)
			}
		}
	}()

	return resultChan, nil
}
