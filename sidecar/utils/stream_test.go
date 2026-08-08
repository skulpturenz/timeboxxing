package utils

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streamPageSize = 2

type streamCall struct {
	page     int
	pageSize int
}

// recordingFetcher serves a fixed table of pages and records what Stream asked for. Pages beyond
// the table report done, which is how a fetcher signals the end.
func recordingFetcher(pages [][]int) (StreamFn[int], func() []streamCall) {
	var mu sync.Mutex

	calls := []streamCall{}

	fetch := func(_ context.Context, page int, pageSize int) ([]int, bool) {
		mu.Lock()
		defer mu.Unlock()

		calls = append(calls, streamCall{page: page, pageSize: pageSize})

		if page >= len(pages) {
			return nil, true
		}

		return pages[page], false
	}

	recorded := func() []streamCall {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(calls)
	}

	return fetch, recorded
}

func TestStream_YieldsEveryPageInOrderAndClosesOnDone(t *testing.T) {
	t.Parallel()

	fetch, _ := recordingFetcher([][]int{{1, 2}, {3, 4}, {5}})

	assert.Equal(t, []int{1, 2, 3, 4, 5}, collect(t, Stream(t.Context(), streamPageSize, fetch)),
		"pages arrive flattened, in order, and the channel closes once the fetcher reports done")
}

func TestStream_AsksForPagesFromZeroWithTheGivenPageSize(t *testing.T) {
	t.Parallel()

	fetch, recorded := recordingFetcher([][]int{{1, 2}, {3, 4}})

	collect(t, Stream(t.Context(), streamPageSize, fetch))

	assert.Equal(t, []streamCall{
		{page: 0, pageSize: streamPageSize},
		{page: 1, pageSize: streamPageSize},
		{page: 2, pageSize: streamPageSize},
	}, recorded(), "paging starts at zero and only advances past a page that did not report done")
}

func TestStream_DiscardsItemsReturnedAlongsideDone(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, page int, _ int) ([]int, bool) {
		if page == 0 {
			return []int{1, 2}, false
		}

		return []int{3, 4}, true
	}

	assert.Equal(t, []int{1, 2}, collect(t, Stream(t.Context(), streamPageSize, fetch)),
		"done discards whatever came with it, so a terminating fetcher returns its last page with done false")
}

func TestStream_FetchesTheNextPageOnlyOnceTheCurrentIsConsumed(t *testing.T) {
	t.Parallel()

	fetch, recorded := recordingFetcher([][]int{{1, 2}, {3, 4}})

	out := Stream(t.Context(), streamPageSize, fetch)

	first, ok := recvWithin(t, out, testTimeout)
	require.True(t, ok)
	require.Equal(t, 1, first)

	assert.Len(t, recorded(), 1, "page one is not fetched while page zero is still sending")

	assert.Equal(t, []int{2, 3, 4}, collect(t, out))
	assert.Len(t, recorded(), 3, "the last call is the one reporting done")
}

func TestStream_CancellationTruncatesTheStreamAndClosesTheChannel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	endless := func(_ context.Context, page int, _ int) ([]int, bool) {
		return []int{page}, false
	}

	out := Stream(ctx, streamPageSize, endless)

	_, ok := recvWithin(t, out, testTimeout)
	require.True(t, ok)

	cancel()

	collect(t, out)

	require.ErrorIs(t, ctx.Err(), context.Canceled,
		"a short read is indistinguishable from an exhausted stream, so consumers check ctx.Err after draining")
}

func TestStream_DoesNotCallTheFetcherWhenTheContextIsAlreadyCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	fetch, recorded := recordingFetcher([][]int{{1, 2}})

	assert.Empty(t, collect(t, Stream(ctx, streamPageSize, fetch)))
	assert.Empty(t, recorded(), "cancellation is checked before the first fetch, so nothing is fetched")
}
