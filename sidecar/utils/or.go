package utils

func Or[T any](predicate func(x T) bool, items ...T) *T {
	for _, v := range items {
		if predicate(v) {
			return &v
		}
	}

	return nil
}
