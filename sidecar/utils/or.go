package utils

import "strings"

func Or[T any](predicate func(x T) bool, items ...T) *T {
	for _, v := range items {
		if predicate(v) {
			return &v
		}
	}

	return nil
}

func IsEmptyString[T string | *string](s T) bool {
	if x, ok := any(s).(*string); ok {
		return x == nil || strings.TrimSpace(*x) == ""
	}

	if x, ok := any(s).(string); ok {
		return strings.TrimSpace(x) == ""
	}

	return false
}

func IsZero[T comparable](x T) bool {
	var zero T

	return zero == x
}
