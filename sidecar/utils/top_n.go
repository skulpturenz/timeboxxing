package utils

import (
	"maps"
	"math"
	"slices"
)

const percentMax = 100.0

type resultTopN[T comparable] []T

func TopN[T comparable](cmp func(a T, b T) int, percentile float64) func([]T) resultTopN[T] {
	return func(xs []T) resultTopN[T] {
		result := []T{}

		if len(xs) == 0 {
			return result
		}

		set := map[T]struct{}{}
		for _, x := range xs {
			set[x] = struct{}{}
		}

		distinct := slices.Collect(maps.Keys(set))

		sorted := append([]T{}, distinct...)
		slices.SortFunc(sorted, func(a T, b T) int {
			return cmp(b, a) // desc
		})

		end := int(math.Ceil(float64(len(sorted)) * (percentile / percentMax)))
		top := sorted[:end]
		topSet := map[T]struct{}{}
		for _, t := range top {
			topSet[t] = struct{}{}
		}

		for _, x := range xs {
			_, ok := topSet[x]
			if !ok {
				continue
			}

			result = append(result, x)
		}

		return result
	}
}

func (t resultTopN[T]) Distinct() []T {
	topSet := map[T]struct{}{}
	for _, t := range t {
		topSet[t] = struct{}{}
	}

	return slices.Collect(maps.Keys(topSet))
}
