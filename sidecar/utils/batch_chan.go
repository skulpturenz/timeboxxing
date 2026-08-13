package utils

import (
	"context"
	"sync"
)

func BatchChan[T any](ctx context.Context, batchSize int, inChan <-chan T, outChans ...chan<- T) func(inChan <-chan any) <-chan any {
	var mu sync.Mutex
	batch := []T{}

	return func(inChan <-chan any) <-chan any {
		result := make(chan any)

		go func() {
			defer close(result)

			for item := range inChan {
				mu.Lock()

				if len(batch) == batchSize {
					result <- batch
				}

				batch = []T{}
				batch = append(batch, item.(T)) //nolint:errcheck // panics. intended

				mu.Unlock()
			}

			if len(batch) != 0 {
				result <- batch
			}
		}()

		return result
	}
}
