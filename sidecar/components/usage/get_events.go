package usage

import (
	"context"
	"fmt"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
)

func (s *Service) GetEvents(ctx context.Context, params GetEventsParams) ([]Event, error) {
	if s == nil || s.transitions == nil {
		return nil, fmt.Errorf("usage transition service is unavailable")
	}

	events, err := s.transitions.GetTransitionEvents(ctx, componentTransitions.GetTransitionEventsParams{})
	if err != nil {
		return nil, err
	}

	usageEvents := make([]Event, 0, len(events))
	for _, event := range events {
		if !eventOverlapsWindow(event, params.Window) {
			continue
		}
		usageEvents = append(usageEvents, eventFromTransition(event))
	}
	if activeEvent, ok := s.activeEvent(ctx, params.Window); ok {
		usageEvents = append(usageEvents, activeEvent)
	}

	return usageEvents, nil
}

func (s *Service) activeEvent(ctx context.Context, window Window) (Event, bool) {
	if s.transitions == nil {
		return Event{}, false
	}
	active, ok, err := s.transitions.GetActiveEvent(ctx)
	if err != nil || !ok {
		return Event{}, false
	}
	return eventFromActiveTransition(active, s.clock(), window)
}
