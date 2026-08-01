package timeline

import (
	"context"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- bounds ----------------------------------------------------------------

// StartedAt is the only bound this query takes, and it is strict: an entry that ended exactly as the
// window opened spent none of its time inside it.
func TestQueryGetTimeline_ReportsEveryEntryFromStartedAtOnwards(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(120)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(180)),
	)

	entries := timelineOf(t, ctx, svcs, ptr(at(60)), 2)

	assert.Equal(t, []string{"Slack", "Zed"}, entryTitles(entries),
		"the entry ending on the opening bound is outside it")
}

// No bound at all reports from the first entry ever recorded.
func TestQueryGetTimeline_StartsAtTheFirstEntryWhenUnbounded(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(120)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(180)),
	)

	entries := timelineOf(t, ctx, svcs, nil, 3)

	assert.Equal(t, []string{"Ghostty", "Slack", "Zed"}, entryTitles(entries))
}

// This is what separates the query from QueryGetTimelineRange: there is no closing bound to fall
// outside of, however far past StartedAt an entry was recorded.
func TestQueryGetTimeline_HasNoClosingBound(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(24*60*60)),
	)

	entries := timelineOf(t, ctx, svcs, ptr(at(0)), 2)

	require.Len(t, entries, 2)
	assert.Equal(t, "Slack", entries[1].Start.Title())
	assert.Equal(t, at(24*60*60), entries[1].End.Timestamp, "a day past the opening bound is still inside")
}

// An entry shorter than the floor is switch noise. The floor is off by default because entries
// chain — dropping one hands its time to its neighbour — so only callers that do not derive
// durations from that adjacency set it.
func TestQueryGetTimeline_FiltersEntriesShorterThanTheFloor(t *testing.T) {
	ctx := context.Background()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(30)),
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(120)),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := &QueryGetTimeline{MinDurationSeconds: 60}
	entries := take(t, utils.Stream(ctx, timelinePageSize, query.Stream(ctx, svcs)), 1)

	assert.Equal(t, []string{"Slack"}, entryTitles(entries),
		"Ghostty held focus for 30s, below the floor; Slack held it for 90s")
}

// --- liveness --------------------------------------------------------------

// This is a live feed, so catching up with the ingest is not the end of it: the stream parks on the
// ingest and reports whatever is recorded next. Only an error ends it — a consumer that wants an
// ending cancels ctx.
func TestQueryGetTimeline_KeepsStreamingPastTheRecordedEntries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
	)

	query := &QueryGetTimeline{StartedAt: ptr(at(0))}
	entries := utils.Stream(ctx, timelinePageSize, query.Stream(ctx, svcs))

	// everything recorded so far, after which the stream has caught up and is waiting rather than
	// finished — nothing here cancels ctx
	assert.Equal(t, []string{"Ghostty"}, entryTitles(take(t, entries, 1)))

	// recorded while that stream is still open. this seed starts its own chain, so it opens an entry
	// at Zed rather than closing the one Slack opened
	seedObservations(t, ctx, svcs,
		appObs("Zed", "dev.zed.Zed", 3, enumscategories.CategoryDevelopment, at(120)),
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(180)),
	)

	assert.Equal(t, []string{"Zed"}, entryTitles(take(t, entries, 1)),
		"the open stream picks up what was written after it caught up")
}

// Cancelling is the only way a consumer ends a live stream, and it must actually end it rather than
// leaking the goroutine behind the channel.
func TestQueryGetTimeline_EndsWhenTheContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svcs := newTestServices(t, ctx)

	seedObservations(t, ctx, svcs,
		appObs("Ghostty", "com.ghostty", 1, enumscategories.CategoryDevelopment, at(0)),
		appObs("Slack", "com.slack", 2, enumscategories.CategoryCommunication, at(60)),
	)

	query := &QueryGetTimeline{StartedAt: ptr(at(0))}
	entries := utils.Stream(ctx, timelinePageSize, query.Stream(ctx, svcs))

	require.Len(t, take(t, entries, 1), 1)

	cancel()

	for {
		select {
		case _, ok := <-entries:
			if !ok {
				return
			}
			// a page already in flight when ctx was cancelled is still delivered; keep draining
		case <-time.After(streamDeadline):
			require.FailNow(t, "the stream outlived its context")
		}
	}
}

// The window is drained across as many keyset pages as it takes, and each entry is reported once.
func TestQueryGetTimeline_ReportsEachEntryOnce(t *testing.T) {
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

	entries := timelineOf(t, ctx, svcs, nil, len(observations)-1)

	require.Len(t, entries, len(observations)-1, "every observation but the last opens an entry")

	seen := map[int64]bool{}
	for _, entry := range entries {
		require.False(t, seen[entry.ID], "entry %d reported twice", entry.ID)
		seen[entry.ID] = true
	}
}

// streamDeadline is how long a test waits for an entry it expects. A live stream that has caught up
// blocks rather than ending, so a broken query would otherwise hang the package instead of failing.
const streamDeadline = 5 * time.Second

// timelineOf opens a live stream, reads the count of entries the test expects, and tears it down.
func timelineOf(t *testing.T, ctx context.Context, svcs *services.Services[any, any], startedAt *time.Time, count int) []models.UsageSeq {
	t.Helper()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := &QueryGetTimeline{StartedAt: startedAt}

	return take(t, utils.Stream(ctx, timelinePageSize, query.Stream(ctx, svcs)), count)
}

// take reads count entries off a live stream. It cannot drain the stream the way collectEntries does
// for the bounded queries: QueryGetTimeline only ends on error, so a consumer bounds itself by what
// it expects. The stream ending early is a failure, not the end of the read.
func take(t *testing.T, entries <-chan models.UsageSeq, count int) []models.UsageSeq {
	t.Helper()

	taken := make([]models.UsageSeq, 0, count)
	for len(taken) < count {
		select {
		case entry, ok := <-entries:
			require.True(t, ok, "the stream ended after %d of %d entries; a live stream ends only on error",
				len(taken), count)

			taken = append(taken, entry)
		case <-time.After(streamDeadline):
			require.FailNowf(t, "timed out", "waited %s for entry %d of %d", streamDeadline, len(taken)+1, count)
		}
	}

	return taken
}
