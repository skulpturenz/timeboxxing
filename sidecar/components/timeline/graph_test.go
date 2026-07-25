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
	assert.Equal(t, []TimeSpan{{at(0), at(10)}, {at(20), at(30)}}, metaA.Intervals)

	metaB, ok := g.GetVertexMeta("B")
	require.True(t, ok)
	assert.Equal(t, 2, metaB.Count)
	assert.Equal(t, 10*time.Second, metaB.Duration)
	assert.Equal(t, []TimeSpan{{at(10), at(20)}}, metaB.Intervals)

	edgeAB, ok := g.GetEdgeMeta("A", "B")
	require.True(t, ok)
	assert.Equal(t, 2, edgeAB.IncomingCount)
	assert.Equal(t, 20*time.Second, edgeAB.IncomingDuration)

	edgeBA, ok := g.GetEdgeMeta("B", "A")
	require.True(t, ok)
	assert.Equal(t, 1, edgeBA.IncomingCount)
	assert.Equal(t, 10*time.Second, edgeBA.IncomingDuration)

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
	assert.Equal(t, 1, edgeAIdle.IncomingCount)

	edgeIdleB, ok := g.GetEdgeMeta("idle", "B")
	require.True(t, ok)
	assert.Equal(t, 1, edgeIdleB.IncomingCount)

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

	var g *ApplicationGraph
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
func mutualCluster(t *testing.T) *ApplicationGraph {
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

// The same set of apps switched in two separate sessions (with an idle gap in between) yields one
// suggestion per session rather than a single merged block. The idle sample joins the SCC via the
// vscode<->idle cycle but is excluded from the labels by the mutual-connection threshold.
func TestGetEntrySuggestions_SameAppsAcrossSessions(t *testing.T) {
	g := GraphFrom(listOf(
		// morning session: vscode <-> terminal
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		// away from keyboard for over a minute
		idleProc(at(50)),
		// afternoon session: the same two apps again
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(180)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(190)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(200)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(210)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(220)),
	))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2, "each session should be its own suggestion")
	for _, s := range suggestions {
		assert.ElementsMatch(t, []string{"vscode", "terminal"}, s.AppIdentifiers)
	}

	// blocks come out ordered by start (flattened is sorted ascending)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(50), suggestions[0].End)
	assert.Equal(t, at(180), suggestions[1].Start)
	assert.Equal(t, at(220), suggestions[1].End)
}

// A single session cycling among three apps several times forms one three-app cluster. vscode is the
// hub the user returns to, so vscode<->terminal and vscode<->chrome are each mutual well above the
// threshold; that pulls all three into the same SCC and the same suggestion. (A pure round-robin
// vscode->terminal->chrome->vscode would instead have only one-way edges and yield no suggestion.)
func TestGetEntrySuggestions_ThreeAppCycleRepeated(t *testing.T) {
	// vscode <-> terminal and vscode <-> chrome, three full rounds
	procs := []ForegroundProcess{}
	second := 0
	push := func(id string, pid int32) {
		procs = append(procs, appProc(id, pid, enumscategories.CategoryDevelopment, at(second)))
		second += 10
	}
	for round := 0; round < 3; round++ {
		push("vscode", 1)
		push("terminal", 2)
		push("vscode", 1)
		push("chrome", 3)
	}
	push("vscode", 1) // trailing sample so the last chrome interval closes

	g := GraphFrom(listOf(procs...))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1, "the repeated three-app session should be one suggestion")
	assert.ElementsMatch(t, []string{"vscode", "terminal", "chrome"}, suggestions[0].AppIdentifiers)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(120), suggestions[0].End) // 13 samples, 10s apart

	// each mutual pair through the hub was traversed once per round, in both directions
	vt, ok := g.GetEdgeMeta("vscode", "terminal")
	require.True(t, ok)
	assert.Equal(t, 3, vt.IncomingCount)
	tv, ok := g.GetEdgeMeta("terminal", "vscode")
	require.True(t, ok)
	assert.Equal(t, 3, tv.IncomingCount)
	vc, ok := g.GetEdgeMeta("vscode", "chrome")
	require.True(t, ok)
	assert.Equal(t, 3, vc.IncomingCount)
	cv, ok := g.GetEdgeMeta("chrome", "vscode")
	require.True(t, ok)
	assert.Equal(t, 3, cv.IncomingCount)
}

// --- GraphChan -------------------------------------------------------------

// GraphChan builds both the metadata maps and the gograph itself from an unsynchronized goroutine, so
// the test is unsafe under -race. To read state without a data race, it relies on the unbuffered
// channel: sending one more (deduplicated) sample only returns once the goroutine has received it,
// which means the previous sample's processing — metadata and graph — is fully committed and the
// goroutine is parked doing no writes.
func TestGraphChan_BuildsGraphAndMeta(t *testing.T) {
	if raceEnabled {
		t.Skip("GraphChan builds the graph from an unsynchronized goroutine; unsafe under -race until it exposes a done signal")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("B", 2, enumscategories.CategoryCommunication, at(10))
	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(20))
	// quiesce: a repeat of the previous sample is deduplicated (no state change); once this send
	// returns the goroutine has finished processing the A@20 sample above.
	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(30))

	// metadata
	metaA, ok := g.GetVertexMeta("A")
	require.True(t, ok)
	assert.Equal(t, 2, metaA.Count)

	metaB, ok := g.GetVertexMeta("B")
	require.True(t, ok)
	assert.Equal(t, 1, metaB.Count)

	_, ok = g.GetEdgeMeta("A", "B")
	assert.True(t, ok)
	_, ok = g.GetEdgeMeta("B", "A")
	assert.True(t, ok)

	// the gograph now mirrors the transitions: both vertices and both directed edges
	assert.Equal(t, uint32(2), g.Graph.Order())

	vA := g.Graph.GetVertexByID("A")
	vB := g.Graph.GetVertexByID("B")
	require.NotNil(t, vA)
	require.NotNil(t, vB)

	assert.NotNil(t, g.Graph.GetEdge(vA, vB), "A->B edge should be in the graph")
	assert.NotNil(t, g.Graph.GetEdge(vB, vA), "B->A edge should be in the graph")
}

// A mutual-switching stream fed through GraphChan produces the same cluster suggestion as GraphFrom.
// GetEntrySuggestions is mutex-protected, so this needs no -race guard; the quiesce send only ensures
// the goroutine has processed every sample before the suggestions are read.
func TestGraphChan_FindsMutualCluster(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	// quiesce: a repeat of the previous sample is deduplicated (no state change); once this send
	// returns the goroutine has finished processing the vscode@40 sample above.
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(50))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1, "the mutual vscode/terminal switching should form one cluster")
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, suggestions[0].AppIdentifiers)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(40), suggestions[0].End)
}

// The multi-cluster scenario from TestGetEntrySuggestions_MultipleClustersExcludeNonClusters, fed
// through GraphChan: two independent mutual clusters each surface as their own suggestion, and the
// one-off apps in between are excluded. GetEntrySuggestions is mutex-protected, so no -race guard is
// needed; the quiesce send only ensures every sample is processed first.
func TestGraphChan_FindsMultipleMutualClusters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	// cluster A: vscode <-> terminal
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	// non-cluster interlude: each visited once, one-way transitions only
	ch <- appProc("slack", 3, enumscategories.CategoryCommunication, at(50))
	ch <- appProc("spotify", 4, enumscategories.CategoryMedia, at(60))
	// cluster B: figma <-> chrome
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(70))
	ch <- appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(80))
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(90))
	ch <- appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(100))
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(110))
	// quiesce: a deduplicated repeat forces the goroutine to finish the figma@110 sample above
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(120))

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

// The same-apps-across-sessions scenario fed through GraphChan: switching between the same two apps in
// two sessions separated by an idle gap yields one suggestion per session.
func TestGraphChan_SameAppsAcrossSessions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	// morning session: vscode <-> terminal
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	// away from keyboard for over a minute
	ch <- idleProc(at(50))
	// afternoon session: the same two apps again
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(180))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(190))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(200))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(210))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(220))
	// quiesce: a deduplicated repeat forces the goroutine to finish the vscode@220 sample above
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(230))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2, "each session should be its own suggestion")
	for _, s := range suggestions {
		assert.ElementsMatch(t, []string{"vscode", "terminal"}, s.AppIdentifiers)
	}

	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(50), suggestions[0].End)
	assert.Equal(t, at(180), suggestions[1].Start)
	assert.Equal(t, at(220), suggestions[1].End)
}

// The repeated three-app session fed through GraphChan: cycling among vscode/terminal/chrome several
// times (with vscode as the hub) forms one three-app cluster, mirroring
// TestGetEntrySuggestions_ThreeAppCycleRepeated.
func TestGraphChan_ThreeAppCycleRepeated(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ForegroundProcess)
	g := GraphChan(ctx, ch)

	second := 0
	send := func(id string, pid int32) {
		ch <- appProc(id, pid, enumscategories.CategoryDevelopment, at(second))
		second += 10
	}
	// vscode <-> terminal and vscode <-> chrome, three full rounds
	for round := 0; round < 3; round++ {
		send("vscode", 1)
		send("terminal", 2)
		send("vscode", 1)
		send("chrome", 3)
	}
	send("vscode", 1) // trailing sample so the last chrome interval closes
	send("vscode", 1) // quiesce: deduplicated repeat forces the goroutine to finish the prior sample

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1, "the repeated three-app session should be one suggestion")
	assert.ElementsMatch(t, []string{"vscode", "terminal", "chrome"}, suggestions[0].AppIdentifiers)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(120), suggestions[0].End)

	vt, ok := g.GetEdgeMeta("vscode", "terminal")
	require.True(t, ok)
	assert.Equal(t, 3, vt.IncomingCount)
	tv, ok := g.GetEdgeMeta("terminal", "vscode")
	require.True(t, ok)
	assert.Equal(t, 3, tv.IncomingCount)
	vc, ok := g.GetEdgeMeta("vscode", "chrome")
	require.True(t, ok)
	assert.Equal(t, 3, vc.IncomingCount)
	cv, ok := g.GetEdgeMeta("chrome", "vscode")
	require.True(t, ok)
	assert.Equal(t, 3, cv.IncomingCount)
}
