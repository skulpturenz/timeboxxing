package timeline

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"github.com/stretchr/testify/require"
)

// base is a fixed, monotonic-clock-free reference so every interval assertion is exact.
var base = time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)

// timelinePageSize is deliberately far smaller than any production page size: a keyset read is only
// interesting once it spans more than one page, and each seeded observation costs a write
// transaction, so the cheapest way to cover paging is to make a page small.
const timelinePageSize = 8

// collectEntries drains a timeline stream into a slice, oldest first.
func collectEntries(ctx context.Context, stream utils.StreamFn[models.UsageSeq]) []models.UsageSeq {
	return slices.Collect(utils.SeqChan(utils.Stream(ctx, timelinePageSize, stream)))
}

func at(seconds int) time.Time {
	return base.Add(time.Duration(seconds) * time.Second)
}

func ptr[T any](v T) *T {
	return new(v)
}

// window spans two offsets from base, in seconds.
func window(fromSeconds int, toSeconds int) utils.TimeSpan {
	return utils.TimeSpan{at(fromSeconds), at(toSeconds)}
}

// appObs builds a non-browser observation. Distinct applications must use distinct identifiers and
// distinct PIDs, since ForegroundProcess.IsEqual keys on PID. Every observation carries a window
// Zero values, declared rather than written as literals so the fixtures below stay readable.
var (
	browserZero     models.Browser
	locationZero    models.Location
	enrichmentsZero models.Enrichments
)

// title and title source, so the columns that hold them stay covered on the round trip.
func appObs(
	name string,
	identifier string,
	pid int64,
	category enumscategories.Category,
	timestamp time.Time,
) models.ForegroundProcess {
	meta := models.AppMetadata{
		FriendlyName: "",
		Description:  "",
		Category:     category,
		IconPath:     "",
		Source:       "",
	}

	return models.ForegroundProcess{
		AppName:       new(name),
		AppIdentifier: new(identifier),
		AppPath:       new("/Applications/" + name + ".app"),
		PID:           new(pid),
		WindowTitle:   new(name + " window"),
		TitleSource:   ptr(models.TitleSourceAX),
		Timestamp:     timestamp,
		Idle:          false,
		Killed:        false,
		Enrichments: models.Enrichments{
			Appmetadata: meta,
			Browser:     browserZero,
			Location:    locationZero,
		},
	}
}

// browserObs builds a browser observation. Every tab of one browser resolves to the same
// application — the event store never records a per-tab identifier.
func browserObs(
	name string,
	identifier string,
	tab string,
	cdpURL string,
	pid int64,
	timestamp time.Time,
) models.ForegroundProcess {
	foregroundProcess := appObs(name, identifier, pid, enumscategories.CategoryWebBrowsing, timestamp)
	foregroundProcess.Enrichments.Browser = models.Browser{
		Vendor:        name,
		Category:      nil,
		AppIdentifier: nil,
		Tab:           tab,
		CdpURL:        cdpURL,
		Domain:        "",
	}

	return foregroundProcess
}

// idleObs builds an idle observation. It carries no identity at all — ForegroundProcess.IsIdle
// asserts exactly that.
func idleObs(timestamp time.Time) models.ForegroundProcess {
	return models.ForegroundProcess{
		AppName:       nil,
		AppIdentifier: nil,
		AppPath:       nil,
		PID:           nil,
		WindowTitle:   nil,
		TitleSource:   nil,
		Timestamp:     timestamp,
		Idle:          true,
		Killed:        false,
		Enrichments:   enrichmentsZero,
	}
}

func newTestServices(ctx context.Context, t *testing.T) *services.Services[any, any] {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		SQLiteVectorExtensionPath: nil,
		DSN:                       db.NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	require.NoError(t, err, "create test database")
	t.Cleanup(func() {
		require.NoError(t, database.Close(), "close test database")
	})

	svcs := services.New()
	db.Register(svcs, database)

	return svcs
}

// seedObservations writes observations through the real ingest command, so the entries under test
// are chained exactly as production writes them: each observation closes the entry the previous one
// opened. Gaps are therefore not representable — to model "nothing happened here", seed an idleObs.
func seedObservations(
	ctx context.Context,
	t *testing.T,
	svcs *services.Services[any, any],
	observations ...models.ForegroundProcess,
) {
	t.Helper()

	var previous *models.ForegroundProcess
	for i := range observations {
		cmd := CommandUpsertForegroundProcess{
			PreviousProcess: previous,
			ActiveProcess:   observations[i],
		}
		require.NoError(t, cmd.Exec(ctx, svcs), "seed observation %d", i)

		previous = &observations[i]
	}
}

// restoreOffset rewrites every stored observation as the same instant expressed at a fixed offset
// from UTC, reproducing the rows written before the ingest normalised to UTC. Those rows are still
// in every existing database, so the window must keep finding them.
func restoreOffset(t *testing.T, svcs *services.Services[any, any], offsetHours int) {
	t.Helper()

	if offsetHours == 0 {
		return
	}

	database, ok := db.FromServices(svcs)
	require.True(t, ok, "database service")

	require.NoError(t, database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		_, err := conn.Exec(
			`UPDATE foreground_processes
			 SET created_at_utc = strftime('%Y-%m-%d %H:%M:%f', created_at_utc, ?) || ?`,
			fmt.Sprintf("%+d hours", offsetHours),
			fmt.Sprintf("%+03d:00", offsetHours),
		)

		return err
	}), "restore stored offset")
}

// entryTitles is the reported titles in order, for asserting shape without pinning every field.
func entryTitles(entries []models.UsageSeq) []string {
	titles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Start == nil {
			titles = append(titles, "")

			continue
		}

		titles = append(titles, entry.Start.Title())
	}

	return titles
}
