package application

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	enumsoperatingsystem "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_operating_system"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/require"
)

func newTestServices(t *testing.T, ctx context.Context) *services.Services[any, any] {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		DSN: db.NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	require.NoError(t, err, "create test database")
	t.Cleanup(func() {
		require.NoError(t, database.Close(), "close test database")
	})

	svcs := services.New()
	db.Register(svcs, database)

	return svcs
}

// seedApplication writes an application and links it to each given category, in the order given,
// through the same upserts ingest uses. It returns the application id the links hang off.
//
// Unlike the timeline component, this seeds the write queries directly rather than through the
// ingest command: an observation would drag in the timeline package, and the timeline package reads
// this one.
func seedApplication(t *testing.T, ctx context.Context, svcs *services.Services[any, any], identifier string, categories ...enumscategories.Category) int64 {
	t.Helper()

	database, ok := db.FromServices(svcs)
	require.True(t, ok, "database service")

	os, err := enumsoperatingsystem.Parse(runtime.GOOS)
	require.NoError(t, err, "parse operating system")

	var applicationId int64
	require.NoError(t, database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
		applicationId, err = q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
			Name:              identifier,
			Identifier:        &identifier,
			OperatingSystemID: int64(os),
		})
		if err != nil {
			return err
		}

		for _, category := range categories {
			// the taxonomy is seeded, so this resolves the existing row by its natural key rather
			// than inserting a new one
			applicationCategoryId, err := q.UpsertApplicationCategory(ctx, writequeries.UpsertApplicationCategoryParams{
				CategoryID: int64(category),
				Code:       category.String(),
				Label:      category.Label(),
			})
			if err != nil {
				return err
			}

			if err := q.UpsertApplicationCategoryMap(ctx, writequeries.UpsertApplicationCategoryMapParams{
				ApplicationID:           applicationId,
				ApplicationCategoriesID: applicationCategoryId,
			}); err != nil {
				return err
			}
		}

		return nil
	}), "seed application %s", identifier)

	return applicationId
}
