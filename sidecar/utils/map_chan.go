package utils

func MapChan[T any, U any](mapper func(T) U) func(<-chan T) <-chan U {
	return func(inChan <-chan T) <-chan U {
		result := make(chan U)

		go func() {
			for item := range inChan {
				result <- mapper(item)
			}
		}()

		return result

	}
}
