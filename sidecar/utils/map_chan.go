package utils

func MapChan[T any, U any](mapper func(T) U) func(inChan <-chan any) <-chan any {
	return func(inChan <-chan any) <-chan any {
		result := make(chan any)

		go func() {
			defer close(result)

			for item := range inChan {
				result <- mapper(item.(T)) //nolint:errcheck // panics. intended
			}
		}()

		return result
	}
}
