package masking

import (
	"context"
	"fmt"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// QueryGetMaskingTables reads both tables whole — they are bounded by the distinct applications and
// domains this install has seen, so unlike every other read here it does not page.
type QueryGetMaskingTables struct{}

func (q QueryGetMaskingTables) Exec(
	ctx context.Context,
	svcs *services.Services[any, any],
) (models.MaskingTables, error) {
	database, ok := db.FromServices(svcs)
	assert.True(ok)
	if !ok {
		return models.MaskingTables{}, fmt.Errorf("get masking tables: database is not registered")
	}

	valueRows, err := database.ReadQuerier.GetMaskedValues(ctx)
	if err != nil {
		return models.MaskingTables{}, fmt.Errorf("get masked values: %w", err)
	}

	values := make(map[models.MaskValue]string, len(valueRows))
	for _, row := range valueRows {
		maskingCategory, parseErr := enumsmaskingcategory.Parse(row.MaskingCategory)
		assert.NoError(parseErr)
		if parseErr != nil {
			return models.MaskingTables{}, parseErr
		}

		values[models.MaskValue{MaskingCategory: maskingCategory, Value: row.Value}] = row.MaskedValue
	}

	categoryRows, err := database.ReadQuerier.GetMaskedCategories(ctx)
	if err != nil {
		return models.MaskingTables{}, fmt.Errorf("get masked categories: %w", err)
	}

	categories := make(map[enumscategories.Category]enumscategories.Category, len(categoryRows))
	for _, row := range categoryRows {
		categories[enumscategories.Category(row.CategoryID)] = enumscategories.Category(row.MaskedCategoryID)
	}

	return models.MaskingTablesFrom(values, categories), nil
}
