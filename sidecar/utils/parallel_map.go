package utils

import (
	"cmp"
	"context"
	"slices"
	"sync"
)

type parallelMapItem[T any] struct {
	idx  int
	item T
}

func ParallelMap[T any, U any](fns ...func(context.Context, T) (U, bool)) func(context.Context, T) ([]U, bool) {
	noop := func(item T) T {
		return item
	}

	return ParallelMapWithClone(noop, fns...)
}

func ParallelMapWithClone[T any, U any](
	clone func(T) T,
	fns ...func(context.Context, T) (U, bool),
) func(context.Context, T) ([]U, bool) {
	return func(ctx context.Context, item T) ([]U, bool) {
		results := make(chan parallelMapItem[U], len(fns))

		var wg sync.WaitGroup
		for idx, fn := range fns {
			wg.Go(func() {
				result, ok := fn(ctx, clone(item))
				if !ok {
					return
				}

				item := parallelMapItem[U]{
					idx:  idx,
					item: result,
				}

				results <- item
			})
		}

		wg.Wait()
		close(results)

		unwrapped := []U{}
		sorted := slices.SortedFunc(SeqChan(results), func(x parallelMapItem[U], y parallelMapItem[U]) int {
			return cmp.Compare(x.idx, y.idx)
		})
		if sorted == nil {
			return unwrapped, false
		}

		for _, v := range sorted {
			unwrapped = append(unwrapped, v.item)
		}

		return unwrapped, len(unwrapped) > 0
	}
}
