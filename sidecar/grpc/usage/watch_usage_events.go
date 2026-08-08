package usage

import (
	"context"
	"log/slog"
	"time"

	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelinemodels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"google.golang.org/grpc"
)

// WatchUsageEvents reports stretches recorded from the window's opening bound onwards and stays
// open: QueryGetTimeline takes no closing bound and never reports itself done, so catching up with
// the ingest does not end the stream — only ctx cancellation (the client going away) does. Keyset
// pagination never yields an entry twice, so nothing has to be deduplicated here.
//
// Entries are published through a one-minute trailing debounce, which is what keeps a sub-minute
// stretch off the timeline. Two things follow from the debounce holding a single latest-wins slot:
// an entry superseded before the next tick is never published — the flicker this exists to suppress
// — and a replayed backlog goes through that same slot, so a client catching up receives the newest
// entry per tick rather than every entry. previous is therefore the previously *published* entry,
// not necessarily the one immediately preceding, which ReasonFor reads as a focus change where the
// dropped neighbour would have made it a tab change.
func (s *Server) WatchUsageEvents(req *usagev1.GetUsageEventsRequest, stream grpc.ServerStreamingServer[usagev1.UsageEvent]) error {
	window, err := windowFromRequest(req)
	if err != nil {
		return err
	}

	// cancelling is the watch's only teardown, so it has to happen on every exit — including the
	// send failure below, which would otherwise leave the producer parked on a full channel
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// the floor runs here, not only downstream: the debounce holds one latest-wins slot, so a
	// sub-minute entry allowed into the channel supersedes whatever real session was waiting there and
	// is then rejected by the filter — publishing neither. Dropping it at the read keeps the slot
	// holding only entries that can survive the filter
	query := &componentTimeline.QueryGetTimeline{
		StartedAt:          &window[0],
		MinDurationSeconds: int64(minEntryDuration.Seconds()),
	}

	var previous *timelinemodels.UsageSeq
	live := utils.Stream(ctx, timelinePageSize, query.Stream(ctx, s.registry))

	ticker := time.NewTicker(minEntryDuration)
	defer ticker.Stop()

	result := utils.PipelineChan[timelinemodels.UsageSeq](utils.TrailingDebounceChan(ctx, ticker.C, live),
		// what the read deliberately lets through: an open entry passes the SQL floor because it has no
		// closing instant yet. Duration clamps that to zero, so it is dropped here rather than in
		// usageSeqToProto. Reading the length off the stretch is also the only correct test —
		// DebouncedItem.Timestamp is when the entry arrived, and the debounce releases it on the next
		// tick, so its age says nothing about how long the focus was held
		utils.FilterChan(func(item utils.DebouncedItem[timelinemodels.UsageSeq]) bool {
			span := item.Value.Span()
			keep := time.Since(item.Value.Start.Timestamp) >= time.Minute

			slog.DebugContext(ctx, "watch: filter", // TODO
				"id", item.Value.ID, "start", span[0], "end", span[1],
				"duration", span.Duration(), "floor", minEntryDuration, "keep", keep)

			return keep
		}),
		utils.MapChan(func(item utils.DebouncedItem[timelinemodels.UsageSeq]) timelinemodels.UsageSeq {
			slog.DebugContext(ctx, "watch: map", "id", item.Value.ID, "arrived", item.Timestamp) // TODO

			return item.Value
		}),
	)

	for entry := range result {
		event, ok := usageSeqToProto(entry, previous, window)

		// ok folds three rejections together — an open entry, and a span that misses the request's
		// window on either side. Overlaps is strict, so a live entry opening after ended_at is dropped
		// here no matter what the pipeline did
		span := entry.Span()
		slog.DebugContext(ctx, "watch: publish", // TODO
			"id", entry.ID, "span", span, "window", window,
			"overlaps", span.Overlaps(window), "open", entry.End == nil, "ok", ok)

		previous = &entry
		if !ok {
			continue
		}

		if err := stream.Send(event); err != nil {
			return err
		}
	}

	return nil
}
