package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The middle of a pipeline is typed `any`, so a stage is an ordinary channel helper wrapped to that
// shape. doubled and evens are the two stages every test below composes.
func doubled(inChan <-chan any) <-chan any {
	return MapChan(func(v any) any {
		return v.(int) * 2
	})(inChan)
}

func evens(inChan <-chan any) <-chan any {
	return FilterChan(func(v any) bool {
		return isEven(v.(int))
	})(inChan)
}

func TestPipelineChan_RunsASingleStageAndTypesTheResult(t *testing.T) {
	t.Parallel()

	source := make(chan any)
	out := PipelineChan[int](source, doubled)

	feed(source, 1, 2, 3)

	assert.Equal(t, []int{2, 4, 6}, collect(t, out), "the tail asserts each value into T on its way out")
}

func TestPipelineChan_RunsStagesInOrder(t *testing.T) {
	t.Parallel()

	stringify := func(inChan <-chan any) <-chan any {
		return MapChan(func(v any) any {
			return strconv.Itoa(v.(int))
		})(inChan)
	}

	source := make(chan any)
	out := PipelineChan[string](source, evens, doubled, stringify)

	feed(source, 1, 2, 3, 4)

	assert.Equal(t, []string{"4", "8"}, collect(t, out),
		"each stage reads the previous stage's output: doubling before the filter would have kept the odd values too")
}

func TestPipelineChan_ClosingTheSourceClosesTheResult(t *testing.T) {
	t.Parallel()

	source := make(chan any)
	out := PipelineChan[int](source, doubled, evens)

	close(source)

	assert.Empty(t, collect(t, out), "the close cascades stage by stage; the source is the only cancellation point")
}

func TestPipelineChan_StagesAreReusableAcrossPipelines(t *testing.T) {
	t.Parallel()

	first := make(chan any)
	second := make(chan any)

	firstOut := PipelineChan[int](first, doubled)
	secondOut := PipelineChan[int](second, doubled)

	feed(first, 1, 2)
	feed(second, 3, 4)

	assert.Equal(t, []int{2, 4}, collect(t, firstOut))
	assert.Equal(t, []int{6, 8}, collect(t, secondOut), "a second pipeline wires the stage its own goroutine")
}
