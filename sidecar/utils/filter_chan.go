package utils

func FilterChan[T any](filter func(T) bool) func(inChan <-chan any) <-chan any {
	return func(inChan <-chan any) <-chan any {
		result := make(chan any)

		go func() {
			defer close(result)

			for item := range inChan {
				if !filter(item.(T)) { //nolint:errcheck // panics. intended
					continue
				}

				result <- item
			}
		}()

		return result
	}
}
