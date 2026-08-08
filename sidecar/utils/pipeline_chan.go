package utils

import "github.com/negrel/assert"

type stage struct {
	in  <-chan any
	out <-chan any
}

func PipelineChan[T any](source <-chan any, pipeline ...func(<-chan any) <-chan any) <-chan T {
	assert.Positive(len(pipeline))

	result := make(chan T)

	stages := []stage{} // instead of plain chan if we need to debug

	for i, c := range pipeline {
		if i == 0 {
			in := source
			out := c(source)
			stages = append(stages, stage{in: in, out: out})
		} else {
			prev := stages[i-1]
			out := c(prev.out)
			stages = append(stages, stage{in: prev.out, out: out})
		}
	}

	go func() {
		defer close(result)

		for v := range stages[len(stages)-1].out {
			result <- v.(T) // panics. intended
		}
	}()

	return result
}
