package timeline

import (
	"context"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- shape -----------------------------------------------------------------

// Each observation closes the entry the previous one opened, so three observations report two
// entries, each running from one observation to the next.
func TestQueryGetTimelineRange_ReportsOneEntryPerFocusedApplication(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(120)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 2)
	assert.Equal(t, []string{"Ghostty", "Slack"}, entryTitles(entries))
	assert.Equal(t, at(0), entries[0].Start.Timestamp)
	assert.Equal(t, at(60), entries[0].End.Timestamp, "the entry ends when the next application takes focus")
	assert.Equal(t, at(60), entries[1].Start.Timestamp)
	assert.Equal(t, at(120), entries[1].End.Timestamp)
}

// A browser entry keeps the tab, which is what the time was actually spent on.
func TestQueryGetTimelineRange_KeepsBrowserEnrichments(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		browserObs("Google Chrome", "com.google.Chrome", "Pull requests", "https://example.com/pulls", 3, at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 1)
	assert.True(t, entries[0].Start.IsBrowser())
	assert.Equal(t, "Pull requests", entries[0].Start.Title())
	assert.Equal(t, "Google Chrome", entries[0].Start.SourceName())
	assert.Equal(t, "https://example.com/pulls", entries[0].Start.Enrichments.Browser.CdpURL)

	// the vendor is persisted, so it reads back as itself rather than the sentinel the converters
	// substitute for a row flagged as a browser but stored without one
	assert.Equal(t, "Google Chrome", entries[0].Start.Enrichments.Browser.Vendor)
}

// The window title and its source are captured per observation rather than per application, so they
// are the two columns that must survive the round trip for an entry to report what was on screen.
func TestQueryGetTimelineRange_KeepsTheWindowTitleAndItsSource(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 1)
	assert.Equal(t, "Ghostty window", utils.Coalesce(entries[0].Start.WindowTitle, ""))
	require.NotNil(t, entries[0].Start.TitleSource, "the title source round trips as its stored code")
	assert.Equal(t, models.TitleSourceAX, *entries[0].Start.TitleSource)

	assert.Equal(t, "Slack window", utils.Coalesce(entries[0].End.WindowTitle, ""),
		"an entry closes on the next observation, which has its own window")
}

// An idle observation has no window, so there is no title source to store. TitleSourceUnknown is not
// a code the read path can parse, so it must be stored as absent rather than as its name.
func TestQueryGetTimelineRange_ReportsNoTitleSourceForIdle(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		idleObs(at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 1)
	assert.Nil(t, entries[0].Start.TitleSource)
	assert.Nil(t, entries[0].Start.WindowTitle)
}

// The category is the one part of the app-metadata enrichment the event store keeps, and it hangs
// off the application rather than the observation — the timeline row carries only the application
// id, so the read looks the classification up and merges it onto both endpoints of an entry.
//
// The graph's productive/unproductive split is derived from this field alone, so an entry that
// reads back as CategoryUnknown is indistinguishable from unproductive time.
func TestQueryGetTimelineRange_ReportsTheApplicationCategoryOnBothEndpoints(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Figma", "com.figma", 3, enumscategories.CategoryGraphicsDesign, at(120)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 2)
	assert.Equal(t, enumscategories.CategoryDevelopment, entries[0].Start.Enrichments.Appmetadata.Category)
	assert.Equal(t, enumscategories.CategoryCommunication, entries[0].End.Enrichments.Appmetadata.Category,
		"an entry closes on the next application, which is classified separately")
	assert.Equal(t, enumscategories.CategoryCommunication, entries[1].Start.Enrichments.Appmetadata.Category)
	assert.Equal(t, enumscategories.CategoryGraphicsDesign, entries[1].End.Enrichments.Appmetadata.Category)
}

// An idle observation has no application, so there is nothing to classify.
func TestQueryGetTimelineRange_ReportsNoCategoryForIdle(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		idleObs(at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 1)
	assert.Equal(t, enumscategories.CategoryUnknown, entries[0].Start.Enrichments.Appmetadata.Category)
	assert.Equal(t, enumscategories.CategoryDevelopment, entries[0].End.Enrichments.Appmetadata.Category)
}

// An idle stretch is recorded, but carries no application identity.
func TestQueryGetTimelineRange_ReportsIdleWithoutAnIdentity(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		idleObs(at(0)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(60)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 200)

	require.Len(t, entries, 1)
	assert.True(t, entries[0].Start.Idle)
	assert.Equal(t, "Idle", entries[0].Start.Title())
	assert.Empty(t, entries[0].Start.ApplicationKey(), "an idle stretch belongs to no application")
}

// --- window bounds ---------------------------------------------------------

// An observation is timestamped in the monitor's local zone, and rows recorded before the ingest
// normalised to UTC kept that offset. A window is always asked for in UTC, so the two are only
// comparable as instants.
//
// Regression: the window bounds were once compared against created_at_utc as text, which compares
// wall clocks. On a +12:00 machine every stretch recorded after local noon read as later than the
// day's closing bound and the timeline came back empty.
func TestQueryGetTimelineRange_FindsObservationsStoredWithANonUTCOffset(t *testing.T) {
	tests := []struct {
		name   string
		offset int
	}{
		{name: "ahead of utc", offset: 12},
		{name: "utc", offset: 0},
		{name: "behind utc", offset: -11},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			svcs := newTestServices(t, ctx)

			seedObservations(t, ctx, svcs,
				appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
				appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
				appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(180)),
			)
			restoreOffset(t, svcs, test.offset)

			entries := rangeOf(t, ctx, svcs, 0, 400)

			require.Len(t, entries, 2, "a UTC window must find observations stored at %+03d:00", test.offset)
			assert.Equal(t, []string{"Ghostty", "Slack"}, entryTitles(entries))
			// compare instants: a stored offset reads back carrying that offset, so two equal instants
			// are not equal values
			assert.True(t, entries[0].Start.Timestamp.Equal(at(0)), "got %v", entries[0].Start.Timestamp)
			assert.True(t, entries[0].End.Timestamp.Equal(at(60)), "got %v", entries[0].End.Timestamp)
			assert.True(t, entries[1].End.Timestamp.Equal(at(180)), "got %v", entries[1].End.Timestamp)
		})
	}
}

// An entry that opens after the window closed belongs to a later window.
//
// Regression: the closing bound was compared against CAST(? AS TIMESTAMP), and sqlite has no
// TIMESTAMP type — the cast is NUMERIC affinity, so the bound collapsed to the year and no row ever
// compared below it. Every request came back empty.
func TestQueryGetTimelineRange_ExcludesEntriesOpeningAfterTheWindowCloses(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(600)),
	)

	entries := rangeOf(t, ctx, svcs, 0, 120)

	require.Len(t, entries, 2)
	assert.Equal(t, []string{"Ghostty", "Slack"}, entryTitles(entries))
	assert.NotContains(t, entryTitles(entries), "Zed", "Zed took focus after the window closed")
}

// The window is half-open and strict at both ends: an entry that ends exactly as the window opens
// spent none of its time inside it, and one that opens exactly as the window closes spent none
// either.
func TestQueryGetTimelineRange_TreatsBothWindowBoundsAsStrict(t *testing.T) {
	tests := []struct {
		name   string
		from   int
		to     int
		titles []string
	}{
		{name: "an entry ending on the opening bound is outside", from: 60, to: 200, titles: []string{"Slack"}},
		{name: "an entry ending after the opening bound is inside", from: 59, to: 200, titles: []string{"Ghostty", "Slack"}},
		{name: "an entry opening on the closing bound is outside", from: 0, to: 60, titles: []string{"Ghostty"}},
		{name: "an entry opening before the closing bound is inside", from: 0, to: 61, titles: []string{"Ghostty", "Slack"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			svcs := newTestServices(t, ctx)

			seedObservations(t, ctx, svcs,
				appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
				appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
				appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(120)),
			)

			entries := rangeOf(t, ctx, svcs, test.from, test.to)

			assert.Equal(t, test.titles, entryTitles(entries))
		})
	}
}

// --- pagination ------------------------------------------------------------

// The window is drained across as many keyset pages as it takes, and each entry is reported once.
func TestQueryGetTimelineRange_DrainsEveryPage(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	observations := make([]models.ForegroundProcess, 0, timelinePageSize+10)
	for i := range cap(observations) {
		name := "App"
		if i%2 == 1 {
			name = "Other"
		}
		observations = append(observations, appObs(name, "com."+name, int64(i%2), enumscategories.CategoryDevelopment, at(i*60)))
	}
	seedObservations(t, ctx, svcs, observations...)

	entries := rangeOf(t, ctx, svcs, 0, len(observations)*60)

	require.Len(t, entries, len(observations)-1, "every observation but the last opens an entry")

	seen := map[int64]bool{}
	for _, entry := range entries {
		require.False(t, seen[entry.ID], "entry %d reported twice", entry.ID)
		seen[entry.ID] = true
	}
}

func rangeOf(t *testing.T, ctx context.Context, svcs *services.Services[any, any], fromSeconds int, toSeconds int) []models.UsageSeq {
	t.Helper()

	span := window(fromSeconds, toSeconds)
	query := &QueryGetTimelineRange{StartedAt: span[0], EndedAt: span[1]}

	return collectEntries(ctx, query.Stream(ctx, svcs))
}
