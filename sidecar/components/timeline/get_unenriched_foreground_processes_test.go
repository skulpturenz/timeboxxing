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

func unenriched(t *testing.T, ctx context.Context, svcs *services.Services[any, any]) []models.ForegroundProcess {
	t.Helper()

	query := &QueryGetUnenrichedForegroundProcesses{}

	return collectProcesses(ctx, query.Stream(ctx, svcs))
}

// A browser observed before the CDP poller resolved its tab is exactly what this backlog is for. It
// must still read back as a browser: ForegroundProcess.IsBrowser reports on the zero-ness of the
// whole Browser enrichment, so an observation that loses every browser field on the round trip is
// indistinguishable from a plain application — and would be re-enriched as one.
func TestQueryGetUnenrichedForegroundProcesses_ReportsATabLessBrowserAsABrowser(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		browserObs("Google Chrome", "com.google.Chrome", "", "ws://localhost:9222", 3, at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	processes := unenriched(t, ctx, svcs)

	require.Len(t, processes, 1, "only the tab-less browser is unenriched")
	assert.True(t, processes[0].IsBrowser())
	assert.Equal(t, "Google Chrome", processes[0].Enrichments.Browser.Vendor)
	assert.Equal(t, "ws://localhost:9222", processes[0].Enrichments.Browser.CdpURL)
	assert.Empty(t, processes[0].Enrichments.Browser.Tab, "the tab is what is missing")
}

// The second arm of the predicate: a public IP recorded without coordinates. It also pins that
// public_ip is written at all — the column is on the observation, not the application.
func TestQueryGetUnenrichedForegroundProcesses_ReportsALocationWithoutCoordinates(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	located := appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0))
	located.Enrichments.Location = models.Location{PublicIP: ptr("203.0.113.7")}

	seedObservations(t, ctx, svcs,
		located,
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
	)

	processes := unenriched(t, ctx, svcs)

	require.Len(t, processes, 1)
	assert.Equal(t, "203.0.113.7", utils.Coalesce(processes[0].Enrichments.Location.PublicIP, ""))
	assert.Nil(t, processes[0].Enrichments.Location.Latitude)
	assert.Nil(t, processes[0].Enrichments.Location.Longitude)
}

// An idle observation has no application, and the query inner-joins applications, so it can never
// surface here however much metadata it is missing.
func TestQueryGetUnenrichedForegroundProcesses_NeverReportsIdle(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		idleObs(at(0)),
		idleObs(at(60)),
	)

	assert.Empty(t, unenriched(t, ctx, svcs))
}
