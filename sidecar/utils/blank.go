package utils

func Blank[T any](x *T) *T {
	if x == nil {
		return nil
	}

	var zero T

	return &zero
}
