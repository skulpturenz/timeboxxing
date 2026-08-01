package applicationgraph

import (
	"slices"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression guard: boundary-adjacent intervals (curr[0] == prev[1]) must merge rather than be
// emitted as one suggestion per interval.
func TestGetEntrySuggestions_MutualClusterProducesOneBlock(t *testing.T) {
	g := mutualCluster(t)

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1)
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(40), suggestions[0].End)
	assert.ElementsMatch(t, []string{"vscode", "terminal"}, suggestions[0].AppIdentifiers)
}

func TestGetEntrySuggestions_MergesAcrossSubMinuteGap(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(10)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(20)),
		appProc("terminal", 2, enumscategories.CategoryDevelopment, at(30)),
		appProc("slack", 3, enumscategories.CategoryCommunication, at(40)), // excluded: switched once
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

func TestGetEntrySuggestions_NoCycleReturnsEmpty(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
		appProc("C", 3, enumscategories.CategoryMedia, at(20)),
	))

	var suggestions []EntrySuggestion
	require.NotPanics(t, func() { suggestions = g.GetEntrySuggestions(at(-1), 2) })
	assert.Empty(t, suggestions)
}

// Regression guard for the empty-flattened bug: an SCC that survives with nothing to suggest must
// return an empty slice, not panic with an index-out-of-range.
func TestGetEntrySuggestions_WeakClusterReturnsEmpty(t *testing.T) {
	t.Run("threshold excludes the only cluster", func(t *testing.T) {
		g := ApplicationGraphFrom(listOf(
			appProc("A", 1, enumscategories.CategoryDevelopment, at(0)),
			appProc("B", 2, enumscategories.CategoryCommunication, at(10)),
			appProc("A", 1, enumscategories.CategoryDevelopment, at(20)),
		))

		require.NotPanics(t, func() {
			assert.Empty(t, g.GetEntrySuggestions(at(-1), 2))
		})
	})

	t.Run("start post-dates all intervals", func(t *testing.T) {
		g := mutualCluster(t)

		require.NotPanics(t, func() {
			assert.Empty(t, g.GetEntrySuggestions(at(3600), 2))
		})
	})
}

// The clusters stay separate SCCs because the timeline never switches back from the second to the
// first.
func TestGetEntrySuggestions_MultipleClustersExcludeNonClusters(t *testing.T) {
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

// The idle sample joins the SCC via the vscode<->idle cycle but is excluded from the labels by the
// mutual-connection threshold.
func TestGetEntrySuggestions_SameAppsAcrossSessions(t *testing.T) {
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

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 2)
	for _, s := range suggestions {
		assert.ElementsMatch(t, []string{"vscode", "terminal"}, s.AppIdentifiers)
	}

	// ordered by start, since flattened is sorted ascending
	assert.Equal(t, at(0), suggestions[0].Start)
	assert.Equal(t, at(50), suggestions[0].End)
	assert.Equal(t, at(180), suggestions[1].Start)
	assert.Equal(t, at(220), suggestions[1].End)
}

// vscode is the hub, so vscode<->terminal and vscode<->chrome are each mutual above the threshold
// and all three land in one SCC. A pure round-robin vscode->terminal->chrome->vscode would have
// only one-way edges and yield no suggestion.
func TestGetEntrySuggestions_ThreeAppCycleRepeated(t *testing.T) {
	procs := []ForegroundProcess{}
	second := 0
	push := func(id string, pid int64) {
		procs = append(procs, appProc(id, pid, enumscategories.CategoryDevelopment, at(second)))
		second += 10
	}
	for range 3 {
		push("vscode", 1)
		push("terminal", 2)
		push("vscode", 1)
		push("chrome", 3)
	}
	push("vscode", 1) // closes the last chrome interval

	g := ApplicationGraphFrom(listOf(procs...))

	suggestions := g.GetEntrySuggestions(at(-1), 2)

	require.Len(t, suggestions, 1)
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
