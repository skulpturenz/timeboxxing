package utils

import (
	"context"
	"log/slog"
)

func FanOutChan[T any](ctx context.Context, inChan <-chan T, outChans ...chan<- T) {
	go func() {
		dropped := map[int]int{}

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-inChan:
				if !ok {
					return
				}

				for i, c := range outChans {
					select {
					case c <- item:
					default:
						dropped[i]++
						slog.ErrorContext(ctx, "fan out chan dropped", "index", i, "drop_count", dropped[i])
					}
				}
			}
		}
	}()
}
