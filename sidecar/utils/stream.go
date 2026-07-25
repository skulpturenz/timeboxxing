package utils

import (
	"context"

	"github.com/negrel/assert"
)

type StreamFn[T any] = func(ctx context.Context, pageSize int) []T

func Stream[T any](ctx context.Context, pageSize int, fn StreamFn[T]) <-chan T {
	assert.Positive(pageSize)

	// buffer size: only want to fetch the next page when the current page has been completely consumed
	// we pull `pageSize` number of items and each send will block if there are no consumers
	result := make(chan T)

	go func() {
		defer close(result)

		page := 0

		for {
			select {
			case <-ctx.Done():
				return
			default:
				items := fn(ctx, pageSize)
				if len(items) == 0 {
					return
				}

				for _, v := range items {
					select {
					case <-ctx.Done():
						return
					case result <- v:
					}
				}

				page += 1
			}
		}
	}()

	return result
}
