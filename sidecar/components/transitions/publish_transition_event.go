package transitions

import (
	"context"
	"fmt"
)

func (s *Service) PublishTransitionEvent(ctx context.Context, params PublishTransitionEventParams) error {
	if s == nil || s.readQuerier == nil {
		return fmt.Errorf("transition event store is unavailable")
	}

	row, err := s.readQuerier.GetTransitionEvent(ctx, params.ID)
	if err != nil {
		return err
	}
	event := eventFromGetTransitionEventRow(row)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, subscriber := range s.subscribers {
		if !eventMatchesFilters(event, subscriber.filters) {
			continue
		}
		select {
		case subscriber.events <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
