package masking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsmaskingcategory "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_masking_category"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// tokenBytes puts a collision against unique_masked_value_token far below anything else that can
// fail here, so one surfaces as an error rather than being retried around.
const tokenBytes = 16

// CommandGenerateMasks is idempotent, which is what makes tokens stable across runs. Concurrent runs
// are safe but not atomic against each other: a category one run loses the race for stays unassigned
// until the next, and models.Mask fails closed on it in the meantime.
type CommandGenerateMasks struct {
	Processes []models.ForegroundProcess
}

func (c CommandGenerateMasks) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	database, ok := db.FromServices(svcs)
	assert.True(ok)
	if !ok {
		return fmt.Errorf("generate masks: database is not registered")
	}

	permutation, err := unassignedPermutation(ctx, database.ReadQuerier)
	if err != nil {
		return err
	}

	values := distinctValues(c.Processes)
	tokens := make([]writequeries.UpsertMaskedValueParams, 0, len(values))
	for _, value := range values {
		token, err := maskedToken(value.MaskingCategory)
		if err != nil {
			return err
		}

		tokens = append(tokens, writequeries.UpsertMaskedValueParams{
			MaskingCategory: value.MaskingCategory.String(),
			Value:           value.Value,
			MaskedValue:     token,
		})
	}

	return database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
		for category, masked := range permutation {
			if err := q.InsertMaskedCategory(ctx, writequeries.InsertMaskedCategoryParams{
				CategoryID:       int64(category),
				MaskedCategoryID: int64(masked),
			}); err != nil {
				return fmt.Errorf("insert masked category %s: %w", category, err)
			}
		}

		for _, token := range tokens {
			if _, err := q.UpsertMaskedValue(ctx, token); err != nil {
				return fmt.Errorf("upsert masked value for %s: %w", token.MaskingCategory, err)
			}
		}

		return nil
	})
}

func distinctValues(processes []models.ForegroundProcess) []models.MaskValue {
	seen := make(map[models.MaskValue]struct{})
	values := make([]models.MaskValue, 0)

	for _, process := range processes {
		for _, value := range process.MaskInputs() {
			if _, ok := seen[value]; ok {
				continue
			}

			seen[value] = struct{}{}
			values = append(values, value)
		}
	}

	return values
}

func unassignedPermutation(
	ctx context.Context,
	querier readqueries.Querier,
) (map[enumscategories.Category]enumscategories.Category, error) {
	taxonomy, err := categoryTaxonomy(ctx, querier)
	if err != nil {
		return nil, err
	}

	assigned, err := querier.GetMaskedCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("get masked categories: %w", err)
	}

	sources := make(map[enumscategories.Category]struct{}, len(taxonomy))
	targets := make(map[enumscategories.Category]struct{}, len(taxonomy))
	for _, category := range taxonomy {
		sources[category] = struct{}{}
		targets[category] = struct{}{}
	}

	for _, row := range assigned {
		delete(sources, enumscategories.Category(row.CategoryID))
		delete(targets, enumscategories.Category(row.MaskedCategoryID))
	}

	free := make([]enumscategories.Category, 0, len(targets))
	for _, category := range taxonomy {
		if _, ok := targets[category]; ok {
			free = append(free, category)
		}
	}

	permutation := make(map[enumscategories.Category]enumscategories.Category, len(sources))
	for _, category := range taxonomy {
		if _, ok := sources[category]; !ok {
			continue
		}

		assert.NotZero(len(free))
		if len(free) == 0 {
			return nil, fmt.Errorf("no stand-in left for category %s", category)
		}

		// a category standing in for itself hides nothing, so it is drawn only as a last resort —
		// greedy assignment can still corner the final source into taking itself
		candidates := free
		if len(free) > 1 {
			candidates = slices.DeleteFunc(slices.Clone(free), func(candidate enumscategories.Category) bool {
				return candidate == category
			})
		}

		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidates))))
		if err != nil {
			return nil, fmt.Errorf("draw stand-in for category %s: %w", category, err)
		}

		masked := candidates[index.Int64()]
		permutation[category] = masked
		free = slices.DeleteFunc(free, func(candidate enumscategories.Category) bool {
			return candidate == masked
		})
	}

	return permutation, nil
}

func categoryTaxonomy(
	ctx context.Context,
	querier readqueries.Querier,
) ([]enumscategories.Category, error) {
	rows, err := querier.GetApplicationCategoryCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get application category codes: %w", err)
	}

	taxonomy := make([]enumscategories.Category, 0, len(rows))
	for _, row := range rows {
		category, err := enumscategories.Parse(row.Code)
		assert.NoError(err)
		if err != nil {
			return nil, err
		}

		if category == enumscategories.CategoryUnknown {
			continue
		}

		taxonomy = append(taxonomy, category)
	}

	return taxonomy, nil
}

func maskedToken(maskingCategory enumsmaskingcategory.MaskingCategory) (string, error) {
	random := make([]byte, tokenBytes)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate mask token: %w", err)
	}

	return maskingCategory.String() + "_" + hex.EncodeToString(random), nil
}
