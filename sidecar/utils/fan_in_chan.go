package utils

import (
	"context"
)

func FanInChan[T any](ctx context.Context, outChan chan<- T, inChans ...<-chan T) {
	for _, inChan := range inChans {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inChan:
					if !ok {
						return
					}

					outChan <- item
				}
			}
		}()
	}
}
