package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FanOutChan hands back no done signal, so every assertion here is over buffered outputs with a
// bounded wait rather than a synchronisation point.

func TestFanOutChan_DeliversEveryItemToEveryOutput(t *testing.T) {
	t.Parallel()

	in := make(chan int)
	first := make(chan int, 3)
	second := make(chan int, 3)

	FanOutChan(t.Context(), in, first, second)

	feed(in, 1, 2, 3)

	assert.Equal(t, []int{1, 2, 3}, recvN(t, first, 3))
	assert.Equal(t, []int{1, 2, 3}, recvN(t, second, 3))
}

func TestFanOutChan_DropsForAnOutputWithNoRoomAndKeepsFeedingTheOthers(t *testing.T) {
	t.Parallel()

	in := make(chan int)
	slow := make(chan int, 1)
	fast := make(chan int, 3)

	FanOutChan(t.Context(), in, slow, fast)

	feed(in, 1, 2, 3)

	assert.Equal(t, []int{1, 2, 3}, recvN(t, fast, 3), "a full output never holds up the others")
	assert.Equal(t, []int{1}, recvN(t, slow, 1))

	requireQuiet(t, slow, "sends are non-blocking, so the items that did not fit were dropped, not queued")
}

func TestFanOutChan_DoesNotCloseTheOutputs(t *testing.T) {
	t.Parallel()

	in := make(chan int)
	out := make(chan int, 1)

	FanOutChan(t.Context(), in, out)

	feed(in, 1)

	assert.Equal(t, []int{1}, recvN(t, out, 1))

	requireQuiet(t, out, "the input closing ends the fan-out, but the outputs belong to the caller")
}

func TestFanOutChan_DrainsTheInputWithNoOutputs(t *testing.T) {
	t.Parallel()

	in := make(chan int)

	FanOutChan(t.Context(), in)

	for i := range 3 {
		sendWithin(t, in, i, testTimeout)
	}
}

func TestFanOutChan_StopsConsumingAfterCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	in := make(chan int)
	out := make(chan int, 1)

	FanOutChan(ctx, in, out)

	cancel()

	require.Eventually(t, func() bool {
		timer := time.NewTimer(settleWait)
		defer timer.Stop()

		select {
		case in <- 1:
			return false
		case <-timer.C:
			return true
		}
	}, testTimeout, settleWait, "a cancelled fan-out stops reading, so the producer is left with nobody to send to")
}
