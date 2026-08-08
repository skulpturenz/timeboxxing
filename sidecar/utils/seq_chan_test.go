package utils

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeqChan_YieldsEveryValueUntilTheChannelCloses(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	assert.Equal(t, []int{1, 2, 3}, slices.Collect(SeqChan(ch)))
}

func TestSeqChan_YieldsNothingForAnAlreadyClosedChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	close(ch)

	assert.Empty(t, slices.Collect(SeqChan(ch)))
}

func TestSeqChan_StoppingEarlyLeavesTheRestOnTheChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	seen := []int{}

	for v := range SeqChan(ch) {
		seen = append(seen, v)

		if v == 2 {
			break
		}
	}

	assert.Equal(t, []int{1, 2}, seen)
	assert.Equal(t, []int{3}, collect(t, ch), "breaking out abandons the remainder rather than draining it")
}
