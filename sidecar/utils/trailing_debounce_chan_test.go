package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tick value is ignored — the ticker is a pure trigger — so these tests hand-feed zero times.
// The emit send is blocking, so the test goroutine reads the output while ticks come from here.

func TestTrailingDebounceChan_ATickEmitsOnlyTheLatestValue(t *testing.T) {
	t.Parallel()

	ticker := make(chan time.Time)
	in := make(chan int)

	out := TrailingDebounceChan(t.Context(), ticker, in)

	sendWithin(t, in, 1, testTimeout)
	sendWithin(t, in, 2, testTimeout)
	sendWithin(t, in, 3, testTimeout)
	sendWithin(t, ticker, time.Time{}, testTimeout)

	item, ok := recvWithin(t, out, testTimeout)
	require.True(t, ok)
	assert.Equal(t, 3, typed[DebouncedItem[int]](t, item).Value,
		"the slot holds one value, so an arrival supersedes the one waiting")

	requireQuiet(t, out, "the superseded values are dropped, never emitted later")
}

func TestTrailingDebounceChan_TimestampIsWhenTheValueArrived(t *testing.T) {
	t.Parallel()

	ticker := make(chan time.Time)
	in := make(chan int)

	out := TrailingDebounceChan(t.Context(), ticker, in)

	sendWithin(t, in, 1, testTimeout)

	arrival := time.Now()

	time.Sleep(2 * settleWait)
	sendWithin(t, ticker, time.Time{}, testTimeout)

	item, ok := recvWithin(t, out, testTimeout)
	require.True(t, ok)

	debounced := typed[DebouncedItem[int]](t, item)

	assert.WithinDuration(t, arrival, debounced.Timestamp, settleWait, "the timestamp is taken as the value arrives")
	assert.GreaterOrEqual(t, time.Since(debounced.Timestamp), 2*settleWait,
		"it is not refreshed when the tick emits it")
}

func TestTrailingDebounceChan_ATickWithNothingPendingEmitsNothing(t *testing.T) {
	t.Parallel()

	ticker := make(chan time.Time)
	in := make(chan int)

	out := TrailingDebounceChan(t.Context(), ticker, in)

	sendWithin(t, ticker, time.Time{}, testTimeout)

	requireQuiet(t, out, "an idle tick is a no-op, not an emission of the zero value")
}

func TestTrailingDebounceChan_FlushesThePendingValueAndClosesAfterTheInputCloses(t *testing.T) {
	t.Parallel()

	ticker := make(chan time.Time)
	in := make(chan int)

	out := TrailingDebounceChan(t.Context(), ticker, in)

	sendWithin(t, in, 1, testTimeout)
	close(in)

	// Termination needs at least one tick after the close, and the goroutine may take the closed
	// input before the tick, so keep ticking until it returns.
	stop := make(chan struct{})
	defer close(stop)

	go func() {
		for {
			select {
			case <-stop:
				return
			case ticker <- time.Time{}:
			}
		}
	}()

	items := collect(t, out)

	require.Len(t, items, 1)
	assert.Equal(t, 1, typed[DebouncedItem[int]](t, items[0]).Value,
		"the pending value is flushed before the goroutine returns")
}

func TestTrailingDebounceChan_CancellationClosesTheOutput(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	ticker := make(chan time.Time)
	in := make(chan int)

	out := TrailingDebounceChan(ctx, ticker, in)

	cancel()

	assert.Empty(t, collect(t, out), "cancellation always closes the output, ticker or not")
}
