package applicationgraph

import (
	"slices"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reading a GraphChan graph needs the goroutine to be quiescent. The channel is unbuffered, so
// sending one more sample that deduplicates against the previous one (same app, later timestamp)
// only returns once the goroutine has received it — by then the sample before it is fully
// committed and the goroutine is parked doing no writes. Each test below ends its stream with such
// a quiesce send.

func TestGraphChan_BuildsGraphAndMeta(t *testing.T) {
	if raceEnabled {
		t.Skip("GraphChan writes the graph from an unsynchronized goroutine; unsafe until it exposes a done signal")
	}

	ch := make(chan ForegroundProcess)
	g := GraphChan(t.Context(), ch)

	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("B", 2, enumscategories.CategoryCommunication, at(10))
	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("A", 1, enumscategories.CategoryDevelopment, at(30)) // quiesce

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

	assert.Equal(t, uint32(2), g.Graph.Order())

	vA := g.Graph.GetVertexByID("A")
	vB := g.Graph.GetVertexByID("B")
	require.NotNil(t, vA)
	require.NotNil(t, vB)

	assert.NotNil(t, g.Graph.GetEdge(vA, vB))
	assert.NotNil(t, g.Graph.GetEdge(vB, vA))
}

// GetEntrySuggestions is mutex-protected, so this and the tests below need no -race guard.
func TestGraphChan_FindsMutualCluster(t *testing.T) {
	ch := make(chan ForegroundProcess)
	g := GraphChan(t.Context(), ch)

	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(50)) // quiesce

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1)
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, suggestions[0].AppIdentifiers)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(40), suggestions[0].End)
}

func TestGraphChan_FindsMultipleMutualClusters(t *testing.T) {
	ch := make(chan ForegroundProcess)
	g := GraphChan(t.Context(), ch)

	// cluster A: vscode <-> terminal
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	// interlude: each visited once, one-way transitions only
	ch <- appProc("slack", 3, enumscategories.CategoryCommunication, at(50))
	ch <- appProc("spotify", 4, enumscategories.CategoryMedia, at(60))
	// cluster B: figma <-> chrome
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(70))
	ch <- appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(80))
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(90))
	ch <- appProc("chrome", 6, enumscategories.CategoryWebBrowsing, at(100))
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(110))
	ch <- appProc("figma", 5, enumscategories.CategoryGraphicsDesign, at(120)) // quiesce

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2)

	for _, s := range suggestions {
		assert.NotContains(t, s.AppIdentifiers, "slack")
		assert.NotContains(t, s.AppIdentifiers, "spotify")
	}

	// matched by content, since Tarjan's component order is not guaranteed
	var coding, design *EntrySuggestion
	for i := range suggestions {
		switch {
		case slices.Contains(suggestions[i].AppIdentifiers, "vscode"):
			coding = &suggestions[i]
		case slices.Contains(suggestions[i].AppIdentifiers, "figma"):
			design = &suggestions[i]
		}
	}

	require.NotNil(t, coding)
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, coding.AppIdentifiers)
	assert.Equal(t, at(0), coding.Start)
	assert.Equal(t, at(50), coding.End)

	require.NotNil(t, design)
	assert.ElementsMatch(t, []string{"figma", "chrome"}, design.AppIdentifiers)
	assert.Equal(t, at(70), design.Start)
	assert.Equal(t, at(110), design.End)
}

func TestGraphChan_SameAppsAcrossSessions(t *testing.T) {
	ch := make(chan ForegroundProcess)
	g := GraphChan(t.Context(), ch)

	// morning session
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40))
	// away from keyboard for over a minute
	ch <- idleProc(at(50))
	// afternoon session, same two apps
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(180))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(190))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(200))
	ch <- appProc("terminal", 2, enumscategories.CategoryDevelopment, at(210))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(220))
	ch <- appProc("vscode", 1, enumscategories.CategoryDevelopment, at(230)) // quiesce

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2)
	for _, s := range suggestions {
		assert.ElementsMatch(t, []string{"vscode", "terminal"}, s.AppIdentifiers)
	}

	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(50), suggestions[0].End)
	assert.Equal(t, at(180), suggestions[1].Start)
	assert.Equal(t, at(220), suggestions[1].End)
}

func TestGraphChan_ThreeAppCycleRepeated(t *testing.T) {
	ch := make(chan ForegroundProcess)
	g := GraphChan(t.Context(), ch)

	second := 0
	send := func(id string, pid int64) {
		ch <- appProc(id, pid, enumscategories.CategoryDevelopment, at(second))
		second += 10
	}
	// vscode <-> terminal and vscode <-> chrome, three full rounds
	for range 3 {
		send("vscode", 1)
		send("terminal", 2)
		send("vscode", 1)
		send("chrome", 3)
	}
	send("vscode", 1) // closes the last chrome interval
	send("vscode", 1) // quiesce

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1)
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
