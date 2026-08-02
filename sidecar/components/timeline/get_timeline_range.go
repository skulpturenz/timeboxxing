package timeline

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/application"
	applicationmodels "github.com/skulpturenz/timeboxxing/sidecar/components/application/models"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/converters"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryGetTimelineRange struct {
	StartedAt          time.Time
	EndedAt            time.Time
	MinDurationSeconds int64
	lastItemID         int64
}

func (q *QueryGetTimelineRange) Stream(
	ctx context.Context,
	svcs *services.Services[any, any],
) utils.StreamFn[models.UsageSeq] {
	database, ok := db.FromServices(svcs)
	assert.True(ok)

	var initialConverter converters.GetTimelineInitialRowConverter
	var finalConverter converters.GetTimelineFinalRowConverter
	var categoriesConverter converters.ApplicationCategoriesConverter

	return func(ctx context.Context, _ int, pageSize int) ([]models.UsageSeq, bool) {
		rows, err := database.ReadQuerier.GetTimeline(ctx, readqueries.GetTimelineParams{
			TimelineId:         q.lastItemID,
			StartedAt:          &q.StartedAt,
			EndedAt:            &q.EndedAt,
			MinDurationSeconds: q.MinDurationSeconds,
			PageSize:           int64(pageSize),
		})
		assert.Condition(func() bool { return ctx.Err() != nil || err == nil })
		if err != nil {
			return nil, true
		}

		if len(rows) == 0 {
			return nil, true // stream ends when there are no more rows
		}

		applicationIDs := map[int64]struct{}{}
		for _, v := range rows {
			for _, applicationID := range []*int64{v.InitialApplicationID, v.FinalApplicationID} {
				if applicationID != nil {
					applicationIDs[*applicationID] = struct{}{}
				}
			}
		}

		appCategoriesCmd := application.QueryGetApplicationCategories{
			ApplicationIDs: slices.Collect(maps.Keys(applicationIDs)),
		}

		appCategories, err := appCategoriesCmd.Exec(ctx, svcs)
		assert.NoError(err)
		if err != nil {
			return nil, true
		}

		result := make([]models.UsageSeq, 0, len(rows))
		for _, v := range rows {
			start := initialConverter.ToForegroundProcess(v)
			if v.InitialApplicationID != nil {
				categoriesConverter.MergeAppMetadata(applicationmodels.ApplicationCategories{
					Categories: appCategories[*v.InitialApplicationID],
				}, &start.Enrichments.Appmetadata)
			}

			// End and Killed stay zero until the entry is closed, just below
			item := models.UsageSeq{
				ID:     v.ID,
				Start:  &start,
				End:    nil,
				Killed: false,
			}

			if v.FinalFpID != nil {
				end := finalConverter.ToForegroundProcess(v)
				if v.FinalApplicationID != nil {
					categoriesConverter.MergeAppMetadata(applicationmodels.ApplicationCategories{
						Categories: appCategories[*v.FinalApplicationID],
					}, &end.Enrichments.Appmetadata)
				}

				item.End = &end
				item.Killed = end.Killed
			}

			result = append(result, item)
		}

		q.lastItemID = rows[len(rows)-1].ID

		return result, false
	}
}
