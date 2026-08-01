package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEdgeMeta_ReturnsMetaForAKnownTransition(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(90)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(150)),
	))

	meta, ok := g.GetEdgeMeta("vscode", "slack")

	require.True(t, ok)
	require.NotNil(t, meta)
	assert.Equal(t, 2, meta.IncomingCount)
	// time spent on vscode before each switch to slack: 60s then 60s
	assert.Equal(t, 120*time.Second, meta.IncomingDuration)
}

func TestGetEdgeMeta_IsDirectional(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(60)),
		appProc("C", 3, enumscategories.CategoryMedia, at(90)),
	))

	_, ok := g.GetEdgeMeta("A", "B")
	assert.True(t, ok)

	meta, ok := g.GetEdgeMeta("B", "A")
	assert.False(t, ok, "the timeline never switched back from B to A")
	assert.Nil(t, meta)
}

func TestGetEdgeMeta_ReportsAMissForAnUnknownTransition(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(60)),
	))

	meta, ok := g.GetEdgeMeta("A", "chrome")
	assert.False(t, ok)
	assert.Nil(t, meta, "a miss must be nil so a caller cannot dereference it by accident")

	empty, ok := GraphFrom(list.New()).GetEdgeMeta("A", "B")
	assert.False(t, ok)
	assert.Nil(t, empty)
}
