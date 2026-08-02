package timeline

import (
	"context"
	"runtime"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsoperatingsystem "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_operating_system"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type CommandUpsertForegroundProcess struct {
	PreviousProcess *models.ForegroundProcess
	ActiveProcess   models.ForegroundProcess
}

// titleSourceCode renders a title source as the code the read path parses back, or nil when there
// is none. TitleSourceUnknown collapses to nil rather than its String(): "unknown" is not one of the
// codes models.ParseTitleSource accepts, so storing it would fail the read.
func titleSourceCode(titleSource *models.TitleSource) *string {
	if titleSource == nil || *titleSource == models.TitleSourceUnknown {
		return nil
	}

	return utils.ZeroNil(titleSource.String())
}

func (c CommandUpsertForegroundProcess) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	database, ok := db.FromServices(svcs)
	assert.True(ok)

	os, err := enumsoperatingsystem.Parse(runtime.GOOS)
	if err != nil {
		return err
	}

	return database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
		var applicationID int64
		if !c.ActiveProcess.IsIdle() {
			category := c.ActiveProcess.Enrichments.Appmetadata.Category

			applicationCategoryID, err := q.UpsertApplicationCategory(ctx, writequeries.UpsertApplicationCategoryParams{
				CategoryID: int64(category),
				Code:       category.String(),
				Label:      category.Label(),
			})
			if err != nil {
				return err
			}

			applicationID, err = q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
				Name:              *c.ActiveProcess.AppName,
				Identifier:        c.ActiveProcess.AppIdentifier,
				OperatingSystemID: int64(os),
				Path:              c.ActiveProcess.AppPath,
			})
			if err != nil {
				return err
			}

			err = q.UpsertApplicationCategoryMap(ctx, writequeries.UpsertApplicationCategoryMapParams{
				ApplicationID:           applicationID,
				ApplicationCategoriesID: applicationCategoryID,
			})
			if err != nil {
				return err
			}
		}

		var pid int64
		if !c.ActiveProcess.IsIdle() {
			pid = *c.ActiveProcess.PID
		}

		id, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
			ApplicationID: utils.ZeroNil(applicationID),
			Pid:           utils.ZeroNil(pid),
			CreatedAtUtc:  c.ActiveProcess.Timestamp.UTC(),
		})
		if err != nil {
			return err
		}

		_, err = q.InsertForegroundProcessMetadata(ctx, writequeries.InsertForegroundProcessMetadataParams{
			ForegroundProcessID: id,
			Browser:             c.ActiveProcess.IsBrowser(),
			BrowserVendor:       utils.ZeroNil(c.ActiveProcess.Enrichments.Browser.Vendor),
			Idle:                c.ActiveProcess.IsIdle(),
			Tab:                 utils.ZeroNil(c.ActiveProcess.Enrichments.Browser.Tab),
			CdpUrl:              utils.ZeroNil(c.ActiveProcess.Enrichments.Browser.CdpURL),
			Latitude:            c.ActiveProcess.Enrichments.Location.Latitude,
			Longitude:           c.ActiveProcess.Enrichments.Location.Longitude,
			PublicIp:            c.ActiveProcess.Enrichments.Location.PublicIP,
			TitleSource:         titleSourceCode(c.ActiveProcess.TitleSource),
			WindowTitle:         c.ActiveProcess.WindowTitle,
		})
		if err != nil {
			return err
		}

		if c.PreviousProcess == nil {
			id, err = q.UpsertTimeline(ctx, writequeries.UpsertTimelineParams{
				InitialForegroundProcessID: &id,
				EndForegroundProcessID:     nil,
			})
			if err != nil {
				return err
			}
		} else {
			id, err = q.UpsertTimeline(ctx, writequeries.UpsertTimelineParams{
				InitialForegroundProcessID: nil,
				EndForegroundProcessID:     &id,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}
