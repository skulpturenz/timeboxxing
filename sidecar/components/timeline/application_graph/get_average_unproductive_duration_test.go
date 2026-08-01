package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
)

// The exact complement of GetAverageProductiveDuration: same per-interval averaging, inverted
// Category.IsProductive check.

func TestGetAverageUnproductiveDuration_AveragesUnproductiveStretches(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
		appProc("notes", 3, enumscategories.CategoryProductivity, at(90)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(210)),
	))

	// slack's single closed interval [60,90]
	assert.Equal(t, 30*time.Second, g.GetAverageUnproductiveDuration())
}

func TestGetAverageUnproductiveDuration_IsZeroWhenEverythingWasProductive(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("notes", 2, enumscategories.CategoryProductivity, at(60)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(90)),
	))

	assert.Zero(t, g.GetAverageUnproductiveDuration())
}

func TestGetAverageUnproductiveDuration_IsZeroForAnEmptyGraph(t *testing.T) {
	assert.Zero(t, ApplicationGraphFrom(list.New()).GetAverageUnproductiveDuration())
}

// Idle time carries no category, and CategoryUnknown is the zero value, so time away from the
// keyboard counts against the unproductive average rather than being skipped.
func TestGetAverageUnproductiveDuration_CountsIdleTime(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		idleProc(at(60)),
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(180)),
	))

	meta, ok := g.GetVertexMeta("idle")
	assert.True(t, ok)
	assert.False(t, meta.Category.IsProductive())

	assert.Equal(t, 120*time.Second, g.GetAverageUnproductiveDuration()) // idle [60,180]
	assert.Equal(t, 60*time.Second, g.GetAverageProductiveDuration())    // vscode [0,60]
}
