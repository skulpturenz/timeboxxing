package utils

func ZeroNil[T comparable](v T) *T {
	if IsZero(v) {
		return nil
	}

	return &v
}
