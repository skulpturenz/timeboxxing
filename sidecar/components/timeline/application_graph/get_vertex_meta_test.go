package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVertexMeta_ReturnsMetaForAKnownLabel(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(90)),
	))

	meta, ok := g.GetVertexMeta("vscode")

	require.True(t, ok)
	require.NotNil(t, meta)
	assert.Equal(t, enumscategories.CategoryDevelopment, meta.Category)
	assert.Equal(t, 2, meta.Count)
	assert.Equal(t, 60*time.Second, meta.Duration)
	assert.Equal(t, []TimeSpan{{at(0), at(60)}}, meta.Intervals)
}

func TestGetVertexMeta_ReportsAMissForAnUnknownLabel(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
	))

	meta, ok := g.GetVertexMeta("chrome")

	assert.False(t, ok)
	assert.Nil(t, meta, "a miss must be nil so a caller cannot dereference it by accident")

	empty, ok := GraphFrom(list.New()).GetVertexMeta("vscode")
	assert.False(t, ok)
	assert.Nil(t, empty)
}

// The key is the graph label, not the raw process identifier.
func TestGetVertexMeta_IsKeyedByGraphLabel(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		idleProc(at(60)),
		browserProc("github.com", enumscategories.CategoryWebBrowsing, 2, at(90)),
	))

	_, ok := g.GetVertexMeta("idle")
	assert.True(t, ok)

	_, ok = g.GetVertexMeta("github.com")
	assert.True(t, ok)

	_, ok = g.GetVertexMeta("com.google.Chrome")
	assert.False(t, ok, "the raw bundle id is not a label")
}
