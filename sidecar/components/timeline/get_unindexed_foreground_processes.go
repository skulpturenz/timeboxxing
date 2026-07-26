package timeline

import (
	"context"
	"reflect"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/converters"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryGetUnindexedForegroundProcesses struct {
	lastItemId int
}

func (q *QueryGetUnindexedForegroundProcesses) Stream(ctx context.Context, svcs *services.Services[any, any]) utils.StreamFn[models.ForegroundProcess] {
	r, ok := services.Get[readqueries.Querier](svcs, reflect.TypeFor[readqueries.Querier]())
	assert.True(ok)

	var converter converters.GetUnindexedForegroundProcessesRowConverter

	return func(ctx context.Context, _ int, pageSize int) ([]models.ForegroundProcess, bool) {
		result := make([]models.ForegroundProcess, pageSize)

		rows, err := r.Unwrap().GetUnindexedForegroundProcesses(ctx, readqueries.GetUnindexedForegroundProcessesParams{
			ForegroundProcessId: int64(q.lastItemId),
			PageSize:            int64(pageSize),
		})
		assert.NoError(err)
		if err != nil {
			return result, true
		}

		for _, v := range rows {
			result = append(result, converter.ToForegroundProcess(v))
		}

		q.lastItemId = int(rows[len(rows)-1].ID)

		return result, len(rows) == 0
	}
}
