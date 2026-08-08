package utils

func FilterChan[T any](filter func(T) bool) func(<-chan T) <-chan T {
	return func(inChan <-chan T) <-chan T {
		result := make(chan T)

		go func() {
			for item := range inChan {
				if !filter(item) {
					continue
				}

				result <- item
			}
		}()

		return result
	}
}
