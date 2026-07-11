package transitions

import (
	"context"
	"fmt"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
)

func (s *Service) GetTransitionEvents(ctx context.Context, params GetTransitionEventsParams) ([]Event, error) {
	if s == nil || s.readQuerier == nil {
		return nil, fmt.Errorf("transition event store is unavailable")
	}

	queryParams := readqueries.GetTransitionEventsParams{}
	if params.Filters.StartedAt != nil {
		queryParams.StartedAt = params.Filters.StartedAt.UTC()
	}
	if params.Filters.EndedAt != nil {
		queryParams.EndedAt = params.Filters.EndedAt.UTC()
	}

	rows, err := s.readQuerier.GetTransitionEvents(ctx, queryParams)
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(rows))
	for i, row := range rows {
		event := eventFromGetTransitionEventsRow(row)
		event.Reason = reasonForTransitionRow(rows, i)
		events = append(events, event)
	}

	return events, nil
}
