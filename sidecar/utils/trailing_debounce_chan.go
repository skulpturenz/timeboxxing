package utils

import (
	"context"
	"log/slog"
	"time"
)

type DebouncedItem[T any] struct {
	Value     T
	Timestamp time.Time
}

func TrailingDebounceChan[T any](
	ctx context.Context,
	ticker <-chan time.Time,
	inChan <-chan T,
) <-chan any {
	// problem: usage timeline shows sessions in 1m granularity. we also have a "Now" indicator
	// if we show a session which is less than 1m, it takes up the 1m block height
	// but now will be within the session block
	// we can't remove the block because it's a now it's here now it's not kind of situation
	result := make(chan any)

	go func() {
		defer close(result)

		var pending T
		timestamp := time.Time{}
		isPending := false

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-inChan:
				if !ok {
					// closed channel will keep hitting this case if we don't nil it
					// nil channels block indefinitely, ignored in select
					inChan = nil
					continue
				}

				slog.DebugContext(ctx, "trailing debounce: arrived", "item", v, "superseded", isPending) // TODO

				pending = v
				timestamp = time.Now()
				isPending = true
			case <-ticker:
				if isPending {
					// build before clearing the slot: Timestamp is when the value arrived, not when it is emitted
					emit := DebouncedItem[T]{Value: pending, Timestamp: timestamp}

					slog.DebugContext(ctx, "trailing debounce: emitting", "item", pending, "held", time.Since(timestamp)) // TODO

					var zero T
					pending = zero
					timestamp = time.Time{}
					isPending = false

					select {
					case <-ctx.Done():
						return
					case result <- emit:
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
