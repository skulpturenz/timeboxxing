package masking

import (
	"context"
	"strings"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandGenerateMasks_IssuesATokenPerValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)
	process := observation("Google Chrome", "com.google.Chrome", "github.com")

	tables := generate(ctx, t, svcs, process)

	for _, value := range process.MaskInputs() {
		token, ok := tables.Values[value]
		require.True(t, ok, "a token was issued for %s", value.MaskingCategory)
		assert.NotEqual(t, value.Value, token, "the token is not the value it stands for")
		assert.True(t, strings.HasPrefix(token, value.MaskingCategory.String()+"_"),
			"the token says which kind of field it stands for")
	}
}

func TestCommandGenerateMasks_IsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)
	process := observation("Google Chrome", "com.google.Chrome", "github.com")

	first := generate(ctx, t, svcs, process)
	second := generate(ctx, t, svcs, process, observation("Ghostty", "com.mitchellh.ghostty", ""))

	for value, token := range first.Values {
		assert.Equal(t, token, second.Values[value], "%s keeps its first token", value.MaskingCategory)
	}
	assert.Equal(t, first.Categories, second.Categories, "the permutation is drawn once")
}

func TestCommandGenerateMasks_ScopesTokensByMaskingCategory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	tables := generate(ctx, t, svcs, observation("Google Chrome", "com.google.Chrome", "github.com"))

	name := tables.Values[models.MaskValue{
		MaskingCategory: enumsmaskingcategory.AppFriendlyName,
		Value:           "Google Chrome",
	}]
	vendor := tables.Values[models.MaskValue{
		MaskingCategory: enumsmaskingcategory.BrowserVendor,
		Value:           "Google Chrome",
	}]

	require.NotEmpty(t, name, "the friendly name has a token")
	assert.NotEqual(t, name, vendor, "the same text in two roles masks to two tokens")
}

func TestCommandGenerateMasks_DrawsACompleteCategoryPermutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	tables := generate(ctx, t, svcs)

	taxonomy := []enumscategories.Category{
		enumscategories.CategoryDevelopment,
		enumscategories.CategoryProductivity,
		enumscategories.CategoryCommunication,
		enumscategories.CategoryWebBrowsing,
		enumscategories.CategoryMedia,
		enumscategories.CategoryGraphicsDesign,
		enumscategories.CategoryGames,
		enumscategories.CategoryUtilities,
		enumscategories.CategoryBusiness,
		enumscategories.CategoryEducation,
		enumscategories.CategorySocial,
		enumscategories.CategorySystem,
		enumscategories.CategoryOther,
	}

	require.Len(t, tables.Categories, len(taxonomy), "every category but unknown has a stand-in")

	standIns := make(map[enumscategories.Category]struct{}, len(taxonomy))
	for _, category := range taxonomy {
		standIn, ok := tables.Categories[category]
		require.True(t, ok, "%s has a stand-in", category)
		assert.NotEqual(t, enumscategories.CategoryUnknown, standIn,
			"unknown is not drawn as a stand-in")

		standIns[standIn] = struct{}{}
	}

	assert.Len(t, standIns, len(taxonomy), "no two categories share a stand-in")
}

func TestCommandGenerateMasks_SurvivesASaturatedPermutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	generate(ctx, t, svcs)
	command := CommandGenerateMasks{Processes: nil}

	assert.NoError(t, command.Exec(ctx, svcs), "a saturated permutation is left alone")
}

func TestCommandGenerateMasks_SkipsAnIdleSample(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	tables := generate(ctx, t, svcs, idleObservation())

	assert.Empty(t, tables.Values, "an idle stretch carries nothing to tokenise")
}
