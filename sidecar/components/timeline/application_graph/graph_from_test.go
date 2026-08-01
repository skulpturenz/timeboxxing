package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	assert.Equal(t, uint32(2), g.Graph.Order())
	// exact only because the rebuild drops a vertex's edges explicitly before removing it;
	// RemoveVertices alone leaks the edge count
	assert.Equal(t, uint32(2), g.Graph.Size(), "A->B and B->A, counted once each")

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
	assert.Equal(t, uint32(2), g.Graph.Size())
}

func TestGraphFrom_BrowserVertexUsesTabIdentifier(t *testing.T) {
	tl := listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		browserProc("github.com", enumscategories.CategoryWebBrowsing, 3, at(20)),
	)

	g := GraphFrom(tl)

	meta, ok := g.GetVertexMeta("github.com")
	require.True(t, ok)
	assert.Equal(t, enumscategories.CategoryWebBrowsing, meta.Category)

	_, ok = g.GetVertexMeta("com.google.Chrome")
	assert.False(t, ok, "the raw chrome bundle id is not a vertex")

	_, ok = g.GetEdgeMeta("terminal", "github.com")
	assert.True(t, ok)
}

// Regression guard: the prev-branch once read curr's browser fields instead of prevProcess's, so a
// non-browser following a browser mislabeled the edge source "com.google.Chrome" — a vertex never
// added.
func TestGraphFrom_BrowserAsPrevLabelsEdgeSource(t *testing.T) {
	tl := listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		browserProc("github.com", enumscategories.CategoryWebBrowsing, 2, at(10)),
		appProc("terminal", 3, enumscategories.CategoryDevelopment, at(20)),
	)

	var g *ApplicationGraph
	require.NotPanics(t, func() { g = GraphFrom(tl) })

	_, hasCorrect := g.GetEdgeMeta("github.com", "terminal")
	assert.True(t, hasCorrect, "edge source should be the browser tab id")

	_, hasBug := g.GetEdgeMeta("com.google.Chrome", "terminal")
	assert.False(t, hasBug)
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
	assert.Empty(t, meta.Intervals, "a lone sample closes no interval")
	assert.Equal(t, uint32(1), single.Graph.Order())
}
