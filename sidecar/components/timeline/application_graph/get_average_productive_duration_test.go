package applicationgraph

import (
	"container/list"
	"testing"
	"time"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
)

// The average is per interval, not per app: an app visited twice contributes two stretches.
// Category.IsProductive covers development, productivity, graphics-design, business and education;
// everything else — including the zero value CategoryUnknown that idle samples carry — does not.

func TestGetAverageProductiveDuration_AveragesProductiveStretches(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("vscode", 1, enumscategories.CategoryDevelopment, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
		appProc("notes", 3, enumscategories.CategoryProductivity, at(90)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(210)),
	))

	// vscode [0,60] is 60s and notes [90,210] is 120s; the trailing slack sample closes no interval
	assert.Equal(t, 90*time.Second, g.GetAverageProductiveDuration())
}

// A real zero rather than a divide by zero over an empty set.
func TestGetAverageProductiveDuration_IsZeroWhenNothingProductiveWasCaptured(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		appProc("slack", 1, enumscategories.CategoryCommunication, at(0)),
		appProc("spotify", 2, enumscategories.CategoryMedia, at(60)),
		appProc("slack", 1, enumscategories.CategoryCommunication, at(90)),
	))

	assert.Zero(t, g.GetAverageProductiveDuration())
}

func TestGetAverageProductiveDuration_IsZeroForAnEmptyGraph(t *testing.T) {
	assert.Zero(t, ApplicationGraphFrom(list.New()).GetAverageProductiveDuration())
}

// A tab is categorised by Enrichments.Browser.Category, not by the browser bundle, so course
// material is productive time even though the vertex belongs to Chrome.
func TestGetAverageProductiveDuration_UsesTheBrowserTabCategory(t *testing.T) {
	g := ApplicationGraphFrom(listOf(
		browserProc("coursera.org", enumscategories.CategoryEducation, 1, at(0)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(60)),
		browserProc("reddit.com", enumscategories.CategorySocial, 3, at(90)),
		appProc("slack", 2, enumscategories.CategoryCommunication, at(150)),
	))

	// only the coursera tab [0,60]
	assert.Equal(t, 60*time.Second, g.GetAverageProductiveDuration())
	// slack [60,90] is 30s and the reddit tab [90,150] is 60s
	assert.Equal(t, 45*time.Second, g.GetAverageUnproductiveDuration())
}
