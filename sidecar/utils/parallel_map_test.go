package utils

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelMap_ReturnsResultsInDeclarationOrder(t *testing.T) {
	t.Parallel()

	released := make(chan struct{})

	declaredFirst := func(_ context.Context, _ int) (string, bool) {
		<-released

		return "first", true
	}

	declaredSecond := func(_ context.Context, _ int) (string, bool) {
		close(released)

		return "second", true
	}

	results, ok := ParallelMap(declaredFirst, declaredSecond)(t.Context(), 0)

	require.True(t, ok)
	assert.Equal(t, []string{"first", "second"}, results,
		"results are re-sorted by declaration index, so completion order does not leak into the output")
}

func TestParallelMap_RunsEveryBranchConcurrently(t *testing.T) {
	t.Parallel()

	const branches = 3

	var started sync.WaitGroup

	started.Add(branches)

	rendezvous := func(_ context.Context, _ int) (int, bool) {
		started.Done()
		started.Wait()

		return 1, true
	}

	done := make(chan []int, 1)

	go func() {
		results, _ := ParallelMap(rendezvous, rendezvous, rendezvous)(t.Context(), 0)
		done <- results
	}()

	results, ok := recvWithin(t, done, testTimeout)

	require.True(t, ok)
	assert.Len(t, results, branches, "no branch can finish until all of them have started")
}

func TestParallelMap_DropsFailedBranches(t *testing.T) {
	t.Parallel()

	succeeds := func(value string) func(context.Context, int) (string, bool) {
		return func(_ context.Context, _ int) (string, bool) {
			return value, true
		}
	}

	fails := func(_ context.Context, _ int) (string, bool) {
		return "unreachable", false
	}

	results, ok := ParallelMap(succeeds("a"), fails, succeeds("c"))(t.Context(), 0)

	assert.True(t, ok, "one success is enough; failures are dropped, never propagated")
	assert.Equal(t, []string{"a", "c"}, results, "the gap left by the failed branch closes up")
}

func TestParallelMap_ReportsFalseWhenEveryBranchFails(t *testing.T) {
	t.Parallel()

	fails := func(_ context.Context, _ int) (string, bool) {
		return "unreachable", false
	}

	results, ok := ParallelMap(fails, fails)(t.Context(), 0)

	assert.False(t, ok)
	assert.NotNil(t, results, "the empty result is an empty slice, never nil")
	assert.Empty(t, results)
}

func TestParallelMap_ReportsFalseWithNoBranches(t *testing.T) {
	t.Parallel()

	results, ok := ParallelMap[int, string]()(t.Context(), 0)

	assert.False(t, ok)
	assert.Empty(t, results)
}

func TestParallelMap_ReturnsAReusableCombinator(t *testing.T) {
	t.Parallel()

	increment := func(_ context.Context, value int) (int, bool) {
		return value + 1, true
	}

	mapper := ParallelMap(increment, increment)

	first, ok := mapper(t.Context(), 1)
	require.True(t, ok)
	assert.Equal(t, []int{2, 2}, first)

	second, ok := mapper(t.Context(), 10)
	require.True(t, ok)
	assert.Equal(t, []int{11, 11}, second, "all the state is per invocation, so the combinator is reusable")
}

func TestParallelMapWithClone_GivesEachBranchItsOwnCopy(t *testing.T) {
	t.Parallel()

	var clones atomic.Int64

	clone := func(m map[string]int) map[string]int {
		clones.Add(1)

		return maps.Clone(m)
	}

	writes := func(key string) func(context.Context, map[string]int) (int, bool) {
		return func(_ context.Context, m map[string]int) (int, bool) {
			m[key] = 1

			return len(m), true
		}
	}

	original := map[string]int{"shared": 0}

	results, ok := ParallelMapWithClone(clone, writes("first"), writes("second"))(t.Context(), original)

	require.True(t, ok)
	assert.Equal(t, []int{2, 2}, results, "each branch sees only its own write on top of the clone")
	assert.Equal(t, map[string]int{"shared": 0}, original, "the caller's value is never handed to a branch")
	assert.Equal(t, int64(2), clones.Load(), "clone runs once per branch, which is what keeps the branches race-free")
}
