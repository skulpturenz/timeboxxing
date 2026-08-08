package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func isEven(v int) bool {
	return v%2 == 0
}

func TestFilterChan_DropsItemsFailingThePredicateAndKeepsOrder(t *testing.T) {
	t.Parallel()

	in := make(chan any)
	out := FilterChan(isEven)(in)

	feed(in, 1, 2, 3, 4, 5, 6)

	assert.Equal(t, []int{2, 4, 6}, typedAll[int](t, collect(t, out)))
}

func TestFilterChan_ClosesTheOutputWhenEveryItemIsDropped(t *testing.T) {
	t.Parallel()

	in := make(chan any)
	out := FilterChan(isEven)(in)

	feed(in, 1, 3, 5)

	assert.Empty(t, collect(t, out), "dropping everything still closes the output when the input closes")
}

func TestFilterChan_ConstructorDrivesIndependentStages(t *testing.T) {
	t.Parallel()

	even := FilterChan(isEven)

	first := make(chan any)
	second := make(chan any)

	firstOut := even(first)
	secondOut := even(second)

	feed(first, 1, 2)
	feed(second, 3, 4)

	assert.Equal(t, []int{2}, typedAll[int](t, collect(t, firstOut)))
	assert.Equal(t, []int{4}, typedAll[int](t, collect(t, secondOut)),
		"each call of the constructor owns its own goroutine")
}
