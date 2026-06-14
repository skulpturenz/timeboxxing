package transitions

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

func (s *Service) GetTransitionEvents(ctx context.Context, params GetTransitionEventsParams) ([]Event, error) {
	queryParams := queries.GetTransitionEventsParams{}
	if params.Filters.StartedAt != nil {
		queryParams.StartedAt = *params.Filters.StartedAt
	}
	if params.Filters.EndedAt != nil {
		queryParams.EndedAt = *params.Filters.EndedAt
	}

	rows, err := s.querier.GetTransitionEvents(ctx, queryParams)
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, eventFromGetTransitionEventsRow(row))
	}

	return events, nil
}
