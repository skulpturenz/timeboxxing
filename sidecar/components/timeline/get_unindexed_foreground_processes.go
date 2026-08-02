package timeline

import (
	"context"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/converters"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryGetUnindexedForegroundProcesses struct {
	lastItemID int64
}

func (q *QueryGetUnindexedForegroundProcesses) Stream(
	ctx context.Context,
	svcs *services.Services[any, any],
) utils.StreamFn[models.ForegroundProcess] {
	database, ok := db.FromServices(svcs)
	assert.True(ok)

	var converter converters.GetUnindexedForegroundProcessesRowConverter

	return func(ctx context.Context, _ int, pageSize int) ([]models.ForegroundProcess, bool) {
		rows, err := database.ReadQuerier.GetUnindexedForegroundProcesses(
			ctx,
			readqueries.GetUnindexedForegroundProcessesParams{
				ForegroundProcessId: q.lastItemID,
				PageSize:            int64(pageSize),
			},
		)
		assert.NoError(err)
		if err != nil {
			return nil, true
		}

		if len(rows) == 0 {
			return nil, true
		}

		result := make([]models.ForegroundProcess, 0, len(rows))

		for _, v := range rows {
			result = append(result, converter.ToForegroundProcess(v))
		}

		q.lastItemID = rows[len(rows)-1].ID

		return result, false
	}
}
