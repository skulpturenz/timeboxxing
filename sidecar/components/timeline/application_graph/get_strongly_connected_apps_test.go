package applicationgraph

import (
	"container/list"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This is the shared engine behind GetFocusScores and GetEntrySuggestions, and the only query here
// that does not take RWMu itself — its callers already hold the read lock. Calling it directly on a
// fully built graph is safe because nothing else is writing.

func TestGetStronglyConnectedApps_ReportsAMutualCluster(t *testing.T) {
	g := mutualCluster(t)

	clusters := g.GetStronglyConnectedApps(2)

	require.Len(t, clusters, 2, "the component is reported under each of its apps")
	require.Contains(t, clusters, "vscode")
	require.Contains(t, clusters, "terminal")
	require.Len(t, clusters["vscode"], 1)

	meta := clusters["vscode"][0]
	assert.Equal(t, []string{"terminal", "vscode"}, meta.AppIdentifiers)
	// vscode [0,10] [20,30] and terminal [10,20] [30,40] are all adjacent, so they merge
	assert.Equal(t, []TimeSpan{{at(0), at(40)}}, meta.Spans)
	assert.Equal(t, map[Edge]int{
		{"vscode", "terminal"}: 2,
		{"terminal", "vscode"}: 2,
	}, meta.EdgeCounts)

	assert.Equal(t, meta, clusters["terminal"][0], "both apps describe the same component")
}

// The threshold separates a working session from an incidental back-and-forth.
func TestGetStronglyConnectedApps_ThresholdExcludesWeakClusters(t *testing.T) {
	g := mutualCluster(t)

	assert.Empty(t, g.GetStronglyConnectedApps(3))
}

func TestGetStronglyConnectedApps_NoCycleReturnsEmpty(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("C", 3, enumscategories.CategoryMedia, at(20)),
	))

	var clusters map[string][]StronglyConnectedEdgesMeta
	require.NotPanics(t, func() { clusters = g.GetStronglyConnectedApps(2) })
	assert.Empty(t, clusters, "every component is a single vertex, which is skipped outright")
}

func TestGetStronglyConnectedApps_EmptyGraphReturnsEmpty(t *testing.T) {
	g := ApplicationGraphFrom(list.New())

	var clusters map[string][]StronglyConnectedEdgesMeta
	require.NotPanics(t, func() { clusters = g.GetStronglyConnectedApps(2) })
	assert.Empty(t, clusters)
}

func TestGetStronglyConnectedApps_SeparatesIndependentClusters(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		// cluster A: vscode <-> terminal
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		// interlude: each visited once, one-way transitions only
		appProc("slack", 3, enumscategories.CategoryCommunication, at(50)),
		appProc("spotify", 4, enumscategories.CategoryMedia, at(60)),
		// cluster B: figma <-> chrome
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(70)),
		appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(80)),
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(90)),
		appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(100)),
		appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(110)),
	))

	clusters := g.GetStronglyConnectedApps(2)

	require.Len(t, clusters, 4, "two components, each reported under both of its apps")
	assert.NotContains(t, clusters, "slack", "a one-way visit is not a cluster")
	assert.NotContains(t, clusters, "spotify")

	require.Len(t, clusters["vscode"], 1)
	coding := clusters["vscode"][0]
	assert.Equal(t, []string{"terminal", "vscode"}, coding.AppIdentifiers)
	assert.Equal(t, []TimeSpan{{at(0), at(50)}}, coding.Spans)
	assert.Equal(t, coding, clusters["terminal"][0])

	require.Len(t, clusters["figma"], 1)
	design := clusters["figma"][0]
	assert.Equal(t, []string{"chrome", "figma"}, design.AppIdentifiers)
	assert.Equal(t, []TimeSpan{{at(70), at(110)}}, design.Spans)
	assert.Equal(t, design, clusters["chrome"][0])
}

// The gap check is what keeps two sessions from collapsing into one block.
func TestGetStronglyConnectedApps_GapOverAMinuteSplitsSpans(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		// morning session
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		// away from keyboard for over a minute
		idleProc(at(50)),
		// afternoon session, same two apps
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(180)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(190)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(200)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(210)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(220)),
	))

	clusters := g.GetStronglyConnectedApps(2)

	require.Contains(t, clusters, "vscode")
	require.Len(t, clusters["vscode"], 1)

	meta := clusters["vscode"][0]
	assert.Equal(t, []string{"terminal", "vscode"}, meta.AppIdentifiers,
		"idle joins the component but is below the mutual threshold, so it is not a label")
	assert.Equal(t, []TimeSpan{{at(0), at(50)}, {at(180), at(220)}}, meta.Spans)
}

// Tarjan's component order, the label set and the edge set are all walked as Go maps, whose
// iteration order is randomised per range. The sorted identifiers are what make the result — and
// any key downstream code derives from it — reproducible.
func TestGetStronglyConnectedApps_IsDeterministic(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("chrome", 3, enumscategories.CategoryWebBrowsing, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(50)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(60)),
		appProc("chrome", 3, enumscategories.CategoryWebBrowsing, at(70)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(80)),
	))

	want := g.GetStronglyConnectedApps(2)
	require.NotEmpty(t, want)

	for i := range 20 {
		assert.Equal(t, want, g.GetStronglyConnectedApps(2), "call %d differed", i)
	}
}
