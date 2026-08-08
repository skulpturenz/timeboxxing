package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// testTimeout bounds a receive or a send the contract says must complete, so a broken helper
	// fails its test instead of hanging the package.
	testTimeout = 2 * time.Second
	// settleWait bounds a receive that must not complete: long enough for a wrong implementation to
	// have produced something, short enough to keep the suite quick.
	settleWait = 100 * time.Millisecond
)

// base is built with [time.Date] so fixture instants carry no monotonic reading and compare with ==.
var base = time.Date(2026, time.August, 8, 9, 0, 0, 0, time.UTC)

// at is the fixture clock: minutes from base.
func at(minutes int) time.Time {
	return base.Add(time.Duration(minutes) * time.Minute)
}

func span(from int, to int) TimeSpan {
	return TimeSpan{at(from), at(to)}
}

// recvWithin receives one value, failing the test rather than blocking forever. The second result
// is the channel's ok, so a closed channel is reported and not mistaken for a timeout.
func recvWithin[T any](t *testing.T, ch <-chan T, within time.Duration) (T, bool) {
	t.Helper()

	timer := time.NewTimer(within)
	defer timer.Stop()

	select {
	case v, ok := <-ch:
		return v, ok
	case <-timer.C:
		var zero T

		require.FailNowf(t, "receive timed out", "nothing arrived within %s", within)

		return zero, false
	}
}

// sendWithin sends one value, failing the test rather than blocking forever.
func sendWithin[T any](t *testing.T, ch chan<- T, v T, within time.Duration) {
	t.Helper()

	timer := time.NewTimer(within)
	defer timer.Stop()

	select {
	case ch <- v:
	case <-timer.C:
		require.FailNowf(t, "send timed out", "no receiver took the value within %s", within)
	}
}

// feed sends every value from a goroutine and closes the channel, leaving the test goroutine free
// to read the far end of whatever the channel feeds.
func feed[T any](ch chan<- T, values ...T) {
	go func() {
		defer close(ch)

		for _, v := range values {
			ch <- v
		}
	}()
}

// collect drains ch until it closes.
func collect[T any](t *testing.T, ch <-chan T) []T {
	t.Helper()

	items := []T{}

	for {
		v, ok := recvWithin(t, ch, testTimeout)
		if !ok {
			return items
		}

		items = append(items, v)
	}
}

// recvN takes exactly n values, for channels the caller does not own and so cannot drain to close.
func recvN[T any](t *testing.T, ch <-chan T, n int) []T {
	t.Helper()

	items := make([]T, 0, n)

	for range n {
		v, ok := recvWithin(t, ch, testTimeout)
		require.Truef(t, ok, "the channel closed before %d values arrived", n)

		items = append(items, v)
	}

	return items
}

// typed re-types a value a stage erased to any, failing the test instead of panicking like the
// pipeline tail does.
func typed[T any](t *testing.T, v any) T {
	t.Helper()

	item, ok := v.(T)
	require.Truef(t, ok, "the stage emitted a %T, not the expected type", v)

	return item
}

// typedAll re-types every value a stage erased to any, so a test asserts on []T and not []any.
func typedAll[T any](t *testing.T, values []any) []T {
	t.Helper()

	items := make([]T, 0, len(values))

	for _, v := range values {
		items = append(items, typed[T](t, v))
	}

	return items
}

// requireQuiet asserts ch produces neither a value nor a close for settleWait.
func requireQuiet[T any](t *testing.T, ch <-chan T, msg string) {
	t.Helper()

	timer := time.NewTimer(settleWait)
	defer timer.Stop()

	select {
	case v, ok := <-ch:
		require.FailNowf(t, msg, "got %v (open: %t) from a channel that should have stayed quiet", v, ok)
	case <-timer.C:
	}
}
