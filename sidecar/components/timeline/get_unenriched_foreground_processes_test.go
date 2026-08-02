package timeline

import (
	"context"
	"slices"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectProcesses drains an observation stream into a slice, oldest first.
func collectProcesses(ctx context.Context, stream utils.StreamFn[models.ForegroundProcess]) []models.ForegroundProcess {
	return slices.Collect(utils.SeqChan(utils.Stream(ctx, timelinePageSize, stream)))
}

func unenriched(ctx context.Context, t *testing.T, svcs *services.Services[any, any]) []models.ForegroundProcess {
	t.Helper()

	query := &QueryGetUnenrichedForegroundProcesses{lastItemID: 0}

	return collectProcesses(ctx, query.Stream(ctx, svcs))
}

// A browser observed before the CDP poller resolved its tab is exactly what this backlog is for. It
// must still read back as a browser: ForegroundProcess.IsBrowser reports on the zero-ness of the
// whole Browser enrichment, so an observation that loses every browser field on the round trip is
// indistinguishable from a plain application — and would be re-enriched as one.
func TestQueryGetUnenrichedForegroundProcesses_ReportsATabLessBrowserAsABrowser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	seedObservations(ctx, t, svcs,
		browserObs("Google Chrome", "com.google.Chrome", "", "ws://localhost:9222", 3, at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	processes := unenriched(ctx, t, svcs)

	require.Len(t, processes, 1, "only the tab-less browser is unenriched")
	assert.True(t, processes[0].IsBrowser())
	assert.Equal(t, "Google Chrome", processes[0].Enrichments.Browser.Vendor)
	assert.Equal(t, "ws://localhost:9222", processes[0].Enrichments.Browser.CdpURL)
	assert.Empty(t, processes[0].Enrichments.Browser.Tab, "the tab is what is missing")
}

// The second arm of the predicate: a public IP recorded without coordinates. It also pins that
// public_ip is written at all — the column is on the observation, not the application.
func TestQueryGetUnenrichedForegroundProcesses_ReportsALocationWithoutCoordinates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	located := appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0))
	located.Enrichments.Location = models.Location{Latitude: nil, Longitude: nil, PublicIP: new("203.0.113.7")}

	seedObservations(ctx, t, svcs,
		located,
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
	)

	processes := unenriched(ctx, t, svcs)

	require.Len(t, processes, 1)
	assert.Equal(t, "203.0.113.7", utils.Coalesce(processes[0].Enrichments.Location.PublicIP, ""))
	assert.Nil(t, processes[0].Enrichments.Location.Latitude)
	assert.Nil(t, processes[0].Enrichments.Location.Longitude)
}

// An idle observation has no application, and the query inner-joins applications, so it can never
// surface here however much metadata it is missing.
func TestQueryGetUnenrichedForegroundProcesses_NeverReportsIdle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svcs := newTestServices(ctx, t)

	seedObservations(ctx, t, svcs,
		idleObs(at(0)),
		idleObs(at(60)),
	)

	assert.Empty(t, unenriched(ctx, t, svcs))
}
