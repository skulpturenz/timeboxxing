package transitions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetActiveEvent returns the current foreground app as the open timeline row (end_foreground_process_id
// IS NULL). The returned Event has its EndedAt unset; callers clip it to "now". ok is false when there
// is no open entry (e.g. before the first observation is recorded).
func (s *Service) GetActiveEvent(ctx context.Context) (Event, bool, error) {
	if s == nil || s.readQuerier == nil {
		return Event{}, false, fmt.Errorf("transition event store is unavailable")
	}

	row, err := s.readQuerier.GetOpenTimelineEvent(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, false, nil
	}
	if err != nil {
		return Event{}, false, err
	}

	return eventFromGetOpenTimelineEventRow(row), true, nil
}
