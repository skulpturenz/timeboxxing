package usage

import (
	"context"
	"slices"

	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelinemodels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func (s *Server) GetUsageEvents(ctx context.Context, req *usagev1.GetUsageEventsRequest) (*usagev1.GetUsageEventsResponse, error) {
	window, err := windowFromRequest(req)
	if err != nil {
		return nil, err
	}

	// the floor matches the one WatchUsageEvents publishes through: a reported stretch runs from one
	// observation to the next, so dropping a sub-minute entry does hand its time to its neighbour —
	// but the two surfaces feed the same timeline, and leaving this at 0 would have a refresh
	// resurrect every stretch the live stream suppressed
	query := &componentTimeline.QueryGetTimelineRange{
		StartedAt:          window[0],
		EndedAt:            window[1],
		MinDurationSeconds: int64(minEntryDuration.Seconds()),
	}
	// the stream ends of its own accord once the window is drained, so the only way this comes back
	// short is a cancelled ctx — which is an error, not an empty window
	entries := slices.Collect(utils.SeqChan(utils.Stream(ctx, timelinePageSize, query.Stream(ctx, s.registry))))
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resp := &usagev1.GetUsageEventsResponse{
		UsageEvents: make([]*usagev1.UsageEvent, 0, len(entries)),
	}
	for i := range entries {
		var previous *timelinemodels.UsageSeq
		if i > 0 {
			previous = &entries[i-1]
		}

		event, ok := usageSeqToProto(entries[i], previous, window)
		if !ok {
			continue
		}

		resp.UsageEvents = append(resp.UsageEvents, event)
	}

	return resp, nil
}
