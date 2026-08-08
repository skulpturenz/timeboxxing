package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapChan_AppliesTheMapperInOrder(t *testing.T) {
	t.Parallel()

	in := make(chan int)
	out := MapChan(strconv.Itoa)(in)

	feed(in, 1, 2, 3)

	assert.Equal(t, []string{"1", "2", "3"}, collect(t, out), "order is preserved and the output closes with the input")
}

func TestMapChan_ClosesTheOutputForAnEmptyInput(t *testing.T) {
	t.Parallel()

	in := make(chan int)
	out := MapChan(strconv.Itoa)(in)

	close(in)

	assert.Empty(t, collect(t, out), "closing the input is the only termination signal the stage has")
}

func TestMapChan_ConstructorDrivesIndependentStages(t *testing.T) {
	t.Parallel()

	double := MapChan(func(v int) int {
		return v * 2
	})

	first := make(chan int)
	second := make(chan int)

	firstOut := double(first)
	secondOut := double(second)

	feed(first, 1, 2)
	feed(second, 3, 4)

	assert.Equal(t, []int{2, 4}, collect(t, firstOut))
	assert.Equal(t, []int{6, 8}, collect(t, secondOut), "each call of the constructor owns its own goroutine")
}
