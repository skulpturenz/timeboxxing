package application

import (
	"context"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The classification ingest wrote is the classification the read reports.
func TestQueryGetApplicationCategories_ReportsTheLinkedCategory(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	applicationId := seedApplication(t, ctx, svcs, "com.ghostty", enumscategories.CategoryDevelopment)

	byApplication, err := QueryGetApplicationCategories{ApplicationIDs: []int64{applicationId}}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.Equal(t, map[int64][]enumscategories.Category{
		applicationId: {enumscategories.CategoryDevelopment},
	}, byApplication)
}

// The whole point of taking a set: a page of timeline rows spans several applications and must cost
// one read, not one per application.
func TestQueryGetApplicationCategories_ReportsEveryApplicationInOneRead(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	ghostty := seedApplication(t, ctx, svcs, "com.ghostty", enumscategories.CategoryDevelopment)
	slack := seedApplication(t, ctx, svcs, "com.slack", enumscategories.CategoryCommunication)
	figma := seedApplication(t, ctx, svcs, "com.figma", enumscategories.CategoryGraphicsDesign)

	byApplication, err := QueryGetApplicationCategories{
		ApplicationIDs: []int64{ghostty, slack, figma},
	}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.Equal(t, map[int64][]enumscategories.Category{
		ghostty: {enumscategories.CategoryDevelopment},
		slack:   {enumscategories.CategoryCommunication},
		figma:   {enumscategories.CategoryGraphicsDesign},
	}, byApplication)
}

// An application nothing has classified yet is absent rather than an error: it has no link rows,
// which is the empty classification. Consumers that carry a single category read that back as
// CategoryUnknown.
func TestQueryGetApplicationCategories_OmitsAnUnclassifiedApplication(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	classified := seedApplication(t, ctx, svcs, "com.ghostty", enumscategories.CategoryDevelopment)
	unclassified := seedApplication(t, ctx, svcs, "com.unclassified")

	byApplication, err := QueryGetApplicationCategories{
		ApplicationIDs: []int64{classified, unclassified},
	}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.NotContains(t, byApplication, unclassified)
	assert.Contains(t, byApplication, classified, "one unclassified application does not hide the rest")
}

// Nor is an application that was never recorded at all.
func TestQueryGetApplicationCategories_OmitsAnUnknownApplication(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	byApplication, err := QueryGetApplicationCategories{ApplicationIDs: []int64{404}}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.Empty(t, byApplication)
}

// A page of nothing but idle observations has no application to ask about, which is not a query.
func TestQueryGetApplicationCategories_ReadsNothingForAnEmptySet(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	byApplication, err := QueryGetApplicationCategories{}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.Empty(t, byApplication)
}

// The read resolves categories by parsing the seeded code, so a category whose code the enum does
// not accept is unreadable. "productivity" and "games" were exactly that.
func TestQueryGetApplicationCategories_ResolvesEveryCategoryInTheTaxonomy(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	expected := map[int64][]enumscategories.Category{}
	applicationIds := []int64{}
	for category := enumscategories.CategoryUnknown; category <= enumscategories.CategoryOther; category++ {
		applicationId := seedApplication(t, ctx, svcs, "com."+category.String(), category)

		expected[applicationId] = []enumscategories.Category{category}
		applicationIds = append(applicationIds, applicationId)
	}

	byApplication, err := QueryGetApplicationCategories{ApplicationIDs: applicationIds}.Exec(ctx, svcs)
	require.NoError(t, err)

	assert.Equal(t, expected, byApplication)
}
