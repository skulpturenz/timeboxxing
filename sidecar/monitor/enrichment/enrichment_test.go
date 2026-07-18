package enrichment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// writeKey returns an enricher that stores value under key. It mutates the map it
// is handed, so if Merge shared one map across the parallel enrichers this would
// trip the runtime's concurrent-map-write detector under -race.
func writeKey(key string, value any) Enricher {
	return func(_ context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		fp.Enrichments[key] = value
		return fp, true
	}
}

func TestMerge_IsolatesConcurrentEnrichers(t *testing.T) {
	fp := monitor.ForegroundProcess{Enrichments: map[string]any{}}

	// Many keys => many goroutines writing at once; the race detector (and the
	// runtime's own concurrent-write check) flags any shared-map mutation.
	enrichers := make([]Enricher, 0, 32)
	for i := range 32 {
		key := string(rune('a' + i))
		enrichers = append(enrichers, writeKey(key, i))
	}

	out, ok := Merge(enrichers...)(context.Background(), fp)
	require.True(t, ok, "expected Merge to report enrichment")
	assert.Len(t, out.Enrichments, 32, "every enricher's key should survive the merge")
	assert.Equal(t, 0, out.Enrichments["a"])
	assert.Equal(t, 31, out.Enrichments[string(rune('a'+31))])
}

func TestMerge_NoEnrichersReportsFalse(t *testing.T) {
	fp := monitor.ForegroundProcess{Enrichments: map[string]any{}}

	noop := func(_ context.Context, p monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		return p, false
	}

	_, ok := Merge(noop, noop)(context.Background(), fp)
	assert.False(t, ok, "no contributing enricher => ok is false")
}
