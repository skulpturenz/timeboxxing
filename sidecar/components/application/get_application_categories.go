package application

import (
	"context"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type QueryGetApplicationCategories struct {
	ApplicationIDs []int64
}

func (q QueryGetApplicationCategories) Exec(
	ctx context.Context,
	svcs *services.Services[any, any],
) (map[int64][]enumscategories.Category, error) {
	appCategories := map[int64][]enumscategories.Category{}

	if len(q.ApplicationIDs) == 0 {
		return appCategories, nil
	}

	database, ok := db.FromServices(svcs)
	assert.True(ok)

	rows, err := database.ReadQuerier.GetApplicationCategories(ctx, q.ApplicationIDs)
	assert.NoError(err)
	if err != nil {
		return appCategories, err
	}

	for _, row := range rows {
		category, err := enumscategories.Parse(row.Code)
		assert.NoError(err)
		if err != nil {
			return appCategories, err
		}

		appCategories[row.ApplicationID] = append(appCategories[row.ApplicationID], category)

		// db allows more than 1 app category for some room
		// app side we're expecting one
		// think linux is the only one where we can have multiple so far and in this case we can pick the most specific one
		assert.Len(appCategories[row.ApplicationID], 1)
	}

	return appCategories, nil
}
