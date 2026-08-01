package applicationgraph

import (
	"container/list"
	"maps"
	"slices"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func focusScoreOf(t *testing.T, scores map[string]float64, identifier string) float64 {
	t.Helper()

	score, ok := scores[identifier]
	require.True(t, ok, "expected %q to be scored, got %v", identifier, scores)

	return score
}

// Regression guard: GetFocusScores used to pair outgoing edges with intervals by index off
// Graph.EdgesOf, and the SCC bookkeeping let whichever app a map range visited first claim the
// shared cycle. Go randomises map iteration, so scores drifted between calls on one graph.
func TestGetFocusScores_IsDeterministic(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(50)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(60)),
		appProc("slack", 3, enumscategories.CategoryCommunication, at(1000)),
	))

	want := g.GetFocusScores(3)
	require.NotEmpty(t, want)

	for i := range 20 {
		assert.Equal(t, want, g.GetFocusScores(3), "call %d differed", i)
	}
}

// Switching between vscode and a terminal is one piece of work, so the block should count as a
// single focused session rather than six short distracted ones.
func TestGetFocusScores_CollapsesMutualCluster(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(40)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(50)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(60)),
		appProc("slack", 3, enumscategories.CategoryCommunication, at(1000)),
	))

	// [0,1000] collapses to one session; vscode keeps its one transition out to slack, so it
	// clears 3 transitions rather than 7
	collapsed := g.GetFocusScores(3)
	assert.InDelta(t, 1.0/3.0, focusScoreOf(t, collapsed, "vscode"), 1e-9)
	assert.InDelta(t, 1.0/2.0, focusScoreOf(t, collapsed, "terminal"), 1e-9)

	// past the threshold there is no cluster, so every switch counts against vscode individually
	split := g.GetFocusScores(4)
	assert.InDelta(t, 1.0/7.0, focusScoreOf(t, split, "vscode"), 1e-9)

	assert.Greater(t, focusScoreOf(t, collapsed, "vscode"), focusScoreOf(t, split, "vscode"),
		"collapsing the cluster should stop counting its switches as distraction")

	assert.NotContains(t, collapsed, "slack",
		"the last sample was never switched away from, so it has no session")
}

// Regression guard: the cycle filter used to append a duration inside the loop over an app's cycle
// blocks, so an app in no cycle at all never ran the body and silently went unscored.
func TestGetFocusScores_ScoresAppsWithoutCycles(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("C", 3, enumscategories.CategoryMedia, at(20)),
	))

	scores := g.GetFocusScores(2)

	// A: one session, one transition out. B: one session, one in and one out.
	assert.InDelta(t, 1.0, focusScoreOf(t, scores, "A"), 1e-9)
	assert.InDelta(t, 0.5, focusScoreOf(t, scores, "B"), 1e-9)
}

// A session is only closed by leaving the app, so the app the timeline ends on has nothing to
// measure.
func TestGetFocusScores_ScoresEveryAppWithASession(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("C", 3, enumscategories.CategoryMedia, at(20)),
		appProc("D", 4, enumscategories.CategoryDevelopment, at(30)),
	))

	scores := g.GetFocusScores(2)

	assert.ElementsMatch(t, []string{"A", "B", "C"}, slices.Collect(maps.Keys(scores)),
		"D is the last sample, so it has no closed session")
}

func TestGetFocusScores_EmptyAndSingle(t *testing.T) {
	assert.Empty(t, ApplicationGraphFrom(list.New()).GetFocusScores(2))

	single := ApplicationGraphFrom(listOf(appProc("A", 1, enumscategories.CategoryDevelopment, base)))
	assert.Empty(t, single.GetFocusScores(2), "a lone sample has no session to score")
}
