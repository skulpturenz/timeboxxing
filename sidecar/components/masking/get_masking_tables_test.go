package masking

import (
	"context"
	"testing"

	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryGetMaskingTables_RoundTripsAMaskedProcess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)
	process := observation("Google Chrome", "com.google.Chrome", "github.com")

	tables := generate(ctx, t, svcs, process)
	masked := process.Mask(tables)

	name, ok := tables.Unmask(enumsmaskingcategory.AppName, *masked.AppName)
	require.True(t, ok, "the app name token resolves")
	assert.Equal(t, "Google Chrome", name, "the token resolves to the real app name")

	domain, ok := tables.Unmask(enumsmaskingcategory.BrowserDomain, masked.Enrichments.Browser.Domain)
	require.True(t, ok, "the domain token resolves")
	assert.Equal(t, "github.com", domain, "the token resolves to the real domain")

	assert.NotEqual(t, process.Enrichments.Appmetadata.Category, masked.Enrichments.Appmetadata.Category,
		"the category stands in for another")
}

func TestQueryGetMaskingTables_IsEmptyBeforeAnythingIsGenerated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	query := QueryGetMaskingTables{}
	tables, err := query.Exec(ctx, svcs)
	require.NoError(t, err, "reading empty tables is not an error")

	assert.Empty(t, tables.Values, "no tokens have been issued")
	assert.Empty(t, tables.Categories, "no permutation has been drawn")

	_, ok := tables.Unmask(enumsmaskingcategory.AppName, "app_name_deadbeef")
	assert.False(t, ok, "nothing resolves out of empty tables")
}

func TestQueryGetMaskingTables_ParsesEveryStoredMaskingCategory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)
	process := observation("Google Chrome", "com.google.Chrome", "github.com")

	tables := generate(ctx, t, svcs, process)

	for value := range tables.Values {
		assert.NotEqual(t, enumsmaskingcategory.Unknown, value.MaskingCategory,
			"every stored code parses back to a real member")
	}

	for category, standIn := range tables.Categories {
		assert.NotEqual(t, enumscategories.CategoryUnknown, category, "unknown is not permuted")
		assert.NotEqual(t, enumscategories.CategoryUnknown, standIn, "unknown is not a stand-in")
	}
}
