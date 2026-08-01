package usage

import (
	"context"

	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelinemodels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"google.golang.org/grpc"
)

// WatchUsageEvents reports every stretch recorded from the window's opening bound onwards and then
// ends: QueryGetTimeline takes no closing bound, but it does report itself done once it has caught
// up with the ingest, so this closes the stream rather than following the ingest from there. Keyset
// pagination never yields an entry twice, so nothing has to be deduplicated here.
func (s *Server) WatchUsageEvents(req *usagev1.GetUsageEventsRequest, stream grpc.ServerStreamingServer[usagev1.UsageEvent]) error {
	window, err := windowFromRequest(req)
	if err != nil {
		return err
	}

	// cancelling is the watch's only teardown, so it has to happen on every exit — including the
	// send failure below, which would otherwise leave the producer parked on a full channel
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	query := &componentTimeline.QueryGetTimeline{StartedAt: &window[0]}

	var previous *timelinemodels.UsageSeq
	for entry := range utils.Stream(ctx, timelinePageSize, query.Stream(ctx, s.registry)) {
		event, ok := usageSeqToProto(entry, previous, window)

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
