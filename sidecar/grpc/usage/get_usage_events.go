package usage

import (
	"context"

	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
)

func (s *Server) GetUsageEvents(ctx context.Context, req *usagev1.GetUsageEventsRequest) (*usagev1.GetUsageEventsResponse, error) {
	window, err := windowFromRequest(req)
	if err != nil {
		return nil, err
	}

	events, err := s.usage.GetEvents(ctx, componentUsage.GetEventsParams{Window: window})
	if err != nil {
		return nil, err
	}

	resp := &usagev1.GetUsageEventsResponse{
		UsageEvents: make([]*usagev1.UsageEvent, 0, len(events)),
	}
	for _, event := range events {
		resp.UsageEvents = append(resp.UsageEvents, eventToProto(event))
	}

	return resp, nil
}
