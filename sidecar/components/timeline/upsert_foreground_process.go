package timeline

import (
	"context"
	"reflect"
	"runtime"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsoperatingsystem "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_operating_system"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type CommandUpsertForegroundProcess struct {
	PreviousProcess *ForegroundProcess
	ActiveProcess   ForegroundProcess
}

func (c CommandUpsertForegroundProcess) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	writeQuerier, ok := services.Get[*db.SerialWriteQuerier](svcs, reflect.TypeFor[db.SerialWriteQuerier]())
	assert.True(ok)

	os, err := enumsoperatingsystem.Parse(runtime.GOOS)
	if err != nil {
		return err
	}

	err = writeQuerier.Unwrap().WriteTx(ctx, func(q *writequeries.Queries) error {
		var applicationId int64
		if !c.ActiveProcess.IsIdle() {
			applicationId, err = q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
				Name:              *c.ActiveProcess.AppName,
				Identifier:        c.ActiveProcess.AppIdentifier,
				OperatingSystemID: int64(os),
				Path:              c.ActiveProcess.AppPath,
			})
			if err != nil {
				return err
			}
		}

		id, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
			ApplicationID: utils.ZeroNil(applicationId),
			Pid:           int64(*c.ActiveProcess.PID),
			CreatedAtUtc:  c.ActiveProcess.Timestamp,
		})
		if err != nil {
			return err
		}

		_, err = q.InsertForegroundProcessMetadata(ctx, writequeries.InsertForegroundProcessMetadataParams{
			ForegroundProcessID: id,
			Browser:             c.ActiveProcess.IsBrowser(),
			Idle:                c.ActiveProcess.IsIdle(),
			Tab:                 utils.ZeroNil(c.ActiveProcess.Enrichments.Browser.Tab),
			CdpUrl:              utils.ZeroNil(c.ActiveProcess.Enrichments.Browser.URL),
			Latitude:            c.ActiveProcess.Enrichments.Location.Latitude,
			Longitude:           c.ActiveProcess.Enrichments.Location.Longitude,
		})
		if err != nil {
			return err
		}

		if c.PreviousProcess == nil {
			id, err = q.UpsertTimeline(ctx, writequeries.UpsertTimelineParams{
				InitialForegroundProcessID: &id,
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

	return err
}
