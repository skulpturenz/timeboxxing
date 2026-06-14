package transitions

import (
	"context"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
)

func (s *Server) GetTransitionEvents(ctx context.Context, req *transitionsv1.GetTransitionEventsRequest) (*transitionsv1.GetTransitionEventsResponse, error) {
	filters, err := filtersFromRequest(req)
	if err != nil {
		return nil, err
	}

	events, err := s.transitions.GetTransitionEvents(ctx, componentTransitions.GetTransitionEventsParams{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	resp := &transitionsv1.GetTransitionEventsResponse{
		TransitionEvents: make([]*transitionsv1.TransitionEvent, 0, len(events)),
	}
	for _, event := range events {
		resp.TransitionEvents = append(resp.TransitionEvents, transitionEventToProto(event))
	}

	return resp, nil
}
