package utils

func Coalesce[T any](x *T, fallback T) T {
	if x == nil {
		return fallback
	}

	return *x
}
