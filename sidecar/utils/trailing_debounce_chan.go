package utils

import (
	"context"
	"time"
)

type item[T any] struct {
	value     T
	timestamp time.Time
}

func TrailingDebounceChan[T any](ctx context.Context, ticker <-chan time.Time, inChan <-chan T) <-chan item[T] {
	// problem: usage timeline shows sessions in 1m granularity. we also have a "Now" indicator
	// if we show a session which is less than 1m, it takes up the 1m block height
	// but now will be within the session block
	// we can't remove the block because it's a now it's here now it's not kind of situation
	result := make(chan item[T])

	go func() {
		defer close(result)

		var pending T
		timestamp := time.Time{}
		isPending := false

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-inChan:
				if !ok {
					// closed channel will keep hitting this case if we don't nil it
					// nil channels block indefinitely, ignored in select
					inChan = nil
					continue
				}

				pending = item
				timestamp = time.Now()
				isPending = true
			case <-ticker:
				if isPending {
					v := pending
					var zero T
					pending = zero
					isPending = false
					timestamp = time.Time{}

					select {
					case <-ctx.Done():
						return
					case result <- item[T]{value: v, timestamp: timestamp}:
					}
				}

				if inChan == nil && !isPending {
					return
				}
			}
		}
	}()

	return result
}
