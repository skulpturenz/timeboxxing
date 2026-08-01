package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The metric walks every unproductive vertex and, for each edge landing on a productive one,
// collects that edge's IncomingDuration — the time spent on the distraction before switching.
//
// Graph.EdgesOf returns incoming edges too, but an incoming edge resolves to the unproductive vertex
// itself and is filtered out by the same IsProductive check, so only outgoing edges contribute.

func TestGetTimeToProductive_ReportsTimeSpentOnADistractionBeforeSwitching(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("slack", 1, enumscategories.CategoryCommunication, at(0)),
		appProc("vscode", 2, enumscategories.CategoryDevelopment, at(60)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(180)),
	))

	// slack -> vscode is the only edge into a productive app: 60s of slack before the switch
	assert.Equal(t, 60*time.Second, g.GetTimeToProductive())
}

func TestGetTimeToProductive_AveragesAcrossDistractions(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("slack", 1, enumscategories.CategoryCommunication, at(0)),
		appProc("vscode", 2, enumscategories.CategoryDevelopment, at(60)),
		appProc("spotify", 3, enumscategories.CategoryMedia, at(120)),
		appProc("figma", 4, enumscategories.CategoryGraphicsDesign, at(300)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(360)),
	))

	// slack -> vscode contributes 60s and spotify -> figma contributes 180s; figma -> slack lands
	// on an unproductive app and is skipped
	assert.Equal(t, 120*time.Second, g.GetTimeToProductive())
}

// IncomingDuration is the running total across every traversal of an edge, and it is collected as a
// single sample, so a distraction returned to repeatedly contributes the sum of its stretches rather
// than one entry per switch.
func TestGetTimeToProductive_SumsRepeatedTraversalsOfOneEdge(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("slack", 1, enumscategories.CategoryCommunication, at(0)),
		appProc("vscode", 2, enumscategories.CategoryDevelopment, at(60)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(120)),
		appProc("vscode", 2, enumscategories.CategoryDevelopment, at(200)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(260)),
	))

	edge, ok := g.GetEdgeMeta("slack", "vscode")
	require.True(t, ok)
	assert.Equal(t, 2, edge.IncomingCount)
	assert.Equal(t, 140*time.Second, edge.IncomingDuration) // 60s then 80s

	// the two switches average 70s each, but the edge is one sample of 140s
	assert.Equal(t, 140*time.Second, g.GetTimeToProductive())
}

func TestGetTimeToProductive_IsZeroWhenAlwaysProductive(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("notes", 2, enumscategories.CategoryProductivity, at(60)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(120)),
	))

	assert.Zero(t, g.GetTimeToProductive(), "nothing unproductive, so no ramp-up to measure")
}

// The same zero as the always-productive case, despite meaning the opposite.
func TestGetTimeToProductive_IsZeroWhenNothingProductiveWasReached(t *testing.T) {
	g := GraphFrom(listOf(
		appProc("slack", 1, enumscategories.CategoryCommunication, at(0)),
		appProc("spotify", 2, enumscategories.CategoryMedia, at(60)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(120)),
	))

	assert.Zero(t, g.GetTimeToProductive())
}

func TestGetTimeToProductive_IsZeroForAnEmptyGraph(t *testing.T) {
	assert.Zero(t, GraphFrom(list.New()).GetTimeToProductive())
}
