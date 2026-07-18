package utils

import "iter"

func SeqChan[T any](ch <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for val := range ch {
			if !yield(val) {
				return
			}
		}
	}
}
