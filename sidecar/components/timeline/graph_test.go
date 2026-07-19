package timeline

import (
	"container/list"
	"context"
	"slices"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// base is a fixed, monotonic-clock-free reference time so interval assertions are deterministic.
var base = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func at(seconds int) time.Time {
	return base.Add(time.Duration(seconds) * time.Second)
}

func ptr[T any](v T) *T { return &v }

// appProc builds a non-browser foreground process; its graph label is the identifier.
// Distinct apps must use distinct PIDs (ForegroundProcess.IsEqual keys on PID).
func appProc(identifier string, pid int32, cat enumscategories.Category, at time.Time) ForegroundProcess {
	name := identifier
	return ForegroundProcess{
		AppName:       &name,
		AppIdentifier: &identifier,
		AppPath:       ptr("/" + identifier),
		PID:           &pid,
		Timestamp:     at,
		Enrichments:   Enrichments{Appmetadata: AppMetadata{Category: cat}},
	}
}

// idleProc builds an idle sample. IsIdle asserts every app field is nil and the timestamp is non-zero.
// Its graph label is "idle".
func idleProc(at time.Time) ForegroundProcess {
	return ForegroundProcess{Idle: true, Timestamp: at}
}

// browserProc builds a browser process; its graph label is the browser tab identifier (tabID),
// and its category comes from Browser.Category.
func browserProc(tabID string, cat enumscategories.Category, pid int32, at time.Time) ForegroundProcess {
	c := cat
	return ForegroundProcess{
		AppIdentifier: ptr("com.google.Chrome"),
		PID:           &pid,
		Timestamp:     at,
		Enrichments: Enrichments{
			Browser: Browser{Browser: "chrome", AppIdentifier: &tabID, Category: &c},
		},
	}
}

func listOf(ps ...ForegroundProcess) *list.List {
	l := list.New()
	for _, p := range ps {
		l.PushBack(p)
	}
	return l
}

// --- GraphFrom -------------------------------------------------------------

func TestGraphFrom_BuildsVerticesEdgesAndCounts(t *testing.T) {
	tl := listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("A", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(30)),
	)

	g := GraphFrom(tl)

	metaA, ok := g.GetVertexMeta("A")
	require.True(t, ok)
	assert.Equal(t, 2, metaA.Count)
	assert.Equal(t, enumscategories.CategoryDevelopment, metaA.Category)
	assert.Equal(t, 20*time.Second, metaA.Duration)
	assert.Equal(t, [][2]time.Time{{at(0), at(10)}, {at(20), at(30)}}, metaA.Intervals)

	metaB, ok := g.GetVertexMeta("B")
	require.True(t, ok)
	assert.Equal(t, 2, metaB.Count)
	assert.Equal(t, 10*time.Second, metaB.Duration)
	assert.Equal(t, [][2]time.Time{{at(10), at(20)}}, metaB.Intervals)

	edgeAB, ok := g.GetEdgeMeta("A", "B")
	require.True(t, ok)
	assert.Equal(t, 2, edgeAB.Count)
	assert.Equal(t, 20*time.Second, edgeAB.Duration)

	edgeBA, ok := g.GetEdgeMeta("B", "A")
	require.True(t, ok)
	assert.Equal(t, 1, edgeBA.Count)
	assert.Equal(t, 10*time.Second, edgeBA.Duration)

	// gograph surface mirrors the metadata.
	assert.Equal(t, uint32(2), g.Graph.Order())
	vA := g.Graph.GetVertexByID("A")
	vB := g.Graph.GetVertexByID("B")
	require.NotNil(t, vA)
	require.NotNil(t, vB)
	assert.Equal(t, float64((20 * time.Second).Milliseconds()), vA.Weight())
	edge := g.Graph.GetEdge(vA, vB)
	require.NotNil(t, edge)
	assert.Equal(t, float64((20 * time.Second).Milliseconds()), edge.Weight())
}

func TestGraphFrom_IdleVertexAndEdges(t *testing.T) {
	tl := listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		idleProc(at(10)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(20)),
	)

	g := GraphFrom(tl)

	_, ok := g.GetVertexMeta("idle")
	assert.True(t, ok, "idle should be its own vertex")

	edgeAIdle, ok := g.GetEdgeMeta("A", "idle")
	require.True(t, ok)
	assert.Equal(t, 1, edgeAIdle.Count)

	edgeIdleB, ok := g.GetEdgeMeta("idle", "B")
	require.True(t, ok)
	assert.Equal(t, 1, edgeIdleB.Count)

	assert.Equal(t, uint32(3), g.Graph.Order())
}

func TestGraphFrom_BrowserVertexUsesTabIdentifier(t *testing.T) {
	tl := listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		browserProc("github.com", enumscategories.CategoryWebBrowsing, 3, at(20)),
	)

	g := GraphFrom(tl)

	meta, ok := g.GetVertexMeta("github.com")
	require.True(t, ok, "browser vertex should be keyed by the tab identifier")
	assert.Equal(t, enumscategories.CategoryWebBrowsing, meta.Category)

	_, ok = g.GetVertexMeta("com.google.Chrome")
	assert.False(t, ok, "should not create a vertex for the raw chrome bundle id")

	_, ok = g.GetEdgeMeta("terminal", "github.com")
	assert.True(t, ok)
}

// Regression guard for the prev-branch browser-identifier bug: an edge leaving a browser must use the
// browser tab id as its source, not the raw chrome bundle id. The prev-branch once read curr's browser
// fields instead of prevProcess's, so a non-browser following a browser mislabeled the source
// "com.google.Chrome" (a vertex never added).
func TestGraphFrom_BrowserAsPrevLabelsEdgeSource(t *testing.T) {
	tl := listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		browserProc("github.com", enumscategories.CategoryWebBrowsing, 2, at(10)),
		appProc("terminal", 3, enumscategories.CategoryDevelopment, at(20)),
	)

	var g TimelineGraph
	require.NotPanics(t, func() { g = GraphFrom(tl) },
		"browser-as-prev must not corrupt the edge source (graph.go:82-83)")

	_, hasCorrect := g.GetEdgeMeta("github.com", "terminal")
	assert.True(t, hasCorrect, "edge source should be the browser tab id")

	_, hasBug := g.GetEdgeMeta("com.google.Chrome", "terminal")
	assert.False(t, hasBug, "edge source must not be the raw chrome bundle id")
}

func TestGraphFrom_EmptyAndSingle(t *testing.T) {
	empty := GraphFrom(list.New())
	assert.Equal(t, uint32(0), empty.Graph.Order())
	_, ok := empty.GetVertexMeta("A")
	assert.False(t, ok)

	single := GraphFrom(listOf(appProc("A", 1, enumscategories.CategoryDevelopment, base)))
	meta, ok := single.GetVertexMeta("A")
	require.True(t, ok)
	assert.Equal(t, 1, meta.Count)
	assert.Empty(t, meta.Intervals)
	assert.Equal(t, uint32(1), single.Graph.Order())
}

// --- GetEntrySuggestions ---------------------------------------------------

// mutualCluster builds a vscode<->terminal graph where each direction is switched twice, so both edges
// pass a threshold of 2 and the pair forms a strongly connected component.
func mutualCluster(t *testing.T) TimelineGraph {
	t.Helper()
	return GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
	))
}

// A contiguous mutual-switching session collapses into a single suggestion. Regression guard for the
// interval-merge bug: boundary-adjacent intervals (curr[0] == prev[1]) must merge rather than be
// emitted as one suggestion per interval.
func TestGetEntrySuggestions_MutualClusterProducesOneBlock(t *testing.T) {
	g := mutualCluster(t)

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1, "a contiguous session should be one suggestion")
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(40), suggestions[0].End)
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, suggestions[0].AppIdentifiers)
}

// A real sub-minute gap (left by a non-cluster app excluded from the labels) is bridged into a single
// suggestion, and the excluded app never appears in a suggestion. Guards against over-merging.
func TestGetEntrySuggestions_MergesAcrossSubMinuteGap(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("slack", 3, enumscategories.CategoryCommunication, at(40)), // excluded: only switched once
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(50)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(60)),
	))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	for _, s := range suggestions {
		assert.NotContains(t, s.AppIdentifiers, "slack", "weakly-connected apps must not be suggested")
	}

	bridged := false
	for _, s := range suggestions {
		// covers both the [30,40] and [50,60] terminal intervals across the slack gap
		if !s.Start.After(at(30)) && !s.End.Before(at(60)) {
			bridged = true
		}
	}
	assert.True(t, bridged, "a sub-minute gap should be merged within one suggestion")
}

// No directed cycle means no SCC of size >= 2, so nothing is suggested.
func TestGetEntrySuggestions_NoCycleReturnsEmpty(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("C", 3, enumscategories.CategoryMedia, at(20)),
	))

	var suggestions []EntrySuggestion
	require.NotPanics(t, func() { suggestions = g.GetEntrySuggestions(at(-1), 2) })
	assert.Empty(t, suggestions)
}

// When an SCC survives but no pair clears the threshold (or start filters every interval) there is
// nothing to suggest. Regression guard for the empty-flattened bug: this must return an empty slice,
// not panic with an index-out-of-range.
func TestGetEntrySuggestions_WeakClusterReturnsEmpty(t *testing.T) {
	t.Run("threshold excludes the only cluster", func(t *testing.T) {
		g := GraphFrom(listOf(
			appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
			appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
			appProc("A", 1, enumscategories.CategoryDevelopment, at(20)),
		))

		require.NotPanics(t, func() {
			assert.Empty(t, g.GetEntrySuggestions(at(-1), 2))
		})
	})

	t.Run("start post-dates all intervals", func(t *testing.T) {
		g := mutualCluster(t) // a real cluster, but every interval is before the requested start

		require.NotPanics(t, func() {
			assert.Empty(t, g.GetEntrySuggestions(at(3600), 2))
		})
	})
}

// Multiple independent mutual clusters in one timeline each yield their own suggestion, while
// foreground processes that never form a mutual cycle (single visits, one-way transitions) are left out
// of every suggestion. The two clusters stay separate SCCs because the timeline never switches back
// from the second cluster to the first.
func TestGetEntrySuggestions_MultipleClustersExcludeNonClusters(t *testing.T) {
	g := GraphFrom(listOf(
		// cluster A: vscode <-> terminal, mutual both directions
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		// non-cluster interlude: each visited once, one-way transitions only
		appProc("slack", 3, enumscategories.CategoryCommunication, at(50)),
		appProc("spotify", 4, enumscategories.CategoryMedia, at(60)),
		// cluster B: figma <-> chrome, mutual both directions
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(70)),
		appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(80)),
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(90)),
		appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(100)),
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(110)),
	))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2, "each mutual cluster should yield exactly one suggestion")

	for _, s := range suggestions {
		assert.NotContains(t, s.AppIdentifiers, "slack", "one-off apps must not be suggested")
		assert.NotContains(t, s.AppIdentifiers, "spotify", "one-off apps must not be suggested")
	}

	// match each suggestion to its cluster regardless of the order Tarjan returns components in
	var coding, design *EntrySuggestion
	for i := range suggestions {
		switch {
		case slices.Contains(suggestions[i].AppIdentifiers, "vscode"):
			coding = &suggestions[i]
		case slices.Contains(suggestions[i].AppIdentifiers, "figma"):
			design = &suggestions[i]
		}
	}

	require.NotNil(t, coding, "expected a vscode/terminal suggestion")
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, coding.AppIdentifiers)
	assert.Equal(t, at(0), coding.Start)
	assert.Equal(t, at(50), coding.End)

	require.NotNil(t, design, "expected a figma/chrome suggestion")
	assert.ElementsMatch(t, []string{"figma", "chrome"}, design.AppIdentifiers)
	assert.Equal(t, at(70), design.Start)
	assert.Equal(t, at(110), design.End)
}

// --- GraphChan -------------------------------------------------------------

// GraphChan spawns an unsynchronized goroutine that writes the meta maps and never populates the
// gograph. This is a best-effort test of the metadata accounting; it is unsafe under -race.
func TestGraphChan_AccumulatesMetaEventually(t *testing.T) {
	if raceEnabled {
		t.Skip("GraphChan writes the meta maps from an unsynchronized goroutine; unsafe under -race until it exposes a done signal")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("B", 2, enumscategories.CategoryCommunication, at(10))
	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(20))

	require.Eventually(t, func() bool {
		meta, ok := g.GetVertexMeta("A")
		return ok && meta.Count == 2
	}, time.Second, 5*time.Millisecond, "goroutine should accumulate vertex metadata")

	metaB, ok := g.GetVertexMeta("B")
	require.True(t, ok)
	assert.Equal(t, 1, metaB.Count)

	_, ok = g.GetEdgeMeta("A", "B")
	assert.True(t, ok)
	_, ok = g.GetEdgeMeta("B", "A")
	assert.True(t, ok)

	// GraphChan only fills metadata; the gograph itself is never populated.
	assert.Equal(t, uint32(0), g.Graph.Order())
}
