package transitions

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
)

func (s *Service) RecordTransitionEvent(ctx context.Context, params RecordTransitionEventParams) (int64, error) {
	if s == nil || s.writeTx == nil {
		return 0, fmt.Errorf("transition event writer is unavailable")
	}

	startedAt, endedAt, useDefaults, err := transitionEventTimestamps(params.StartedAt, params.EndedAt)
	if err != nil {
		return 0, err
	}

	var eventID int64
	if err := s.writeTx.WriteTx(ctx, func(q *writequeries.Queries) error {
		appName := strings.TrimSpace(params.ApplicationName)
		applicationID := sql.NullInt64{}

		if !params.Idle && appName != "" {
			id, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
				Name:               appName,
				PlatformIdentifier: nullString(params.ApplicationIdentifier),
				Path:               nullString(params.ApplicationPath),
			})
			if err != nil {
				return fmt.Errorf("upsert application %q: %w", appName, err)
			}
			applicationID = sql.NullInt64{Int64: id, Valid: true}
		}

		var err error
		if useDefaults {
			eventID, err = q.CreateTransitionEventNow(ctx, writequeries.CreateTransitionEventNowParams{
				ApplicationID: applicationID,
				Reason:        params.Reason,
			})
		} else {
			eventID, err = q.CreateTransitionEvent(ctx, writequeries.CreateTransitionEventParams{
				ApplicationID: applicationID,
				Reason:        params.Reason,
				StartedAt:     startedAt,
				EndedAt:       endedAt,
			})
		}
		if err != nil {
			return fmt.Errorf("create transition event: %w", err)
		}

		if err := q.CreateTransitionEventMetadata(ctx, writequeries.CreateTransitionEventMetadataParams{
			TransitionEventID: eventID,
			Browser:           params.Browser,
			Tab:               params.Tab,
			Idle:              params.Idle,
			CdpUrl:            params.CDPURL,
			Pid:               nullPID(params.PID),
		}); err != nil {
			return fmt.Errorf("create transition event metadata: %w", err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	if err := s.PublishTransitionEvent(ctx, PublishTransitionEventParams{ID: eventID}); err != nil {
		return 0, fmt.Errorf("publish transition event: %w", err)
	}

	return eventID, nil
}

func transitionEventTimestamps(startedAt time.Time, endedAt time.Time) (time.Time, time.Time, bool, error) {
	startedAtZero := startedAt.IsZero()
	endedAtZero := endedAt.IsZero()
	if startedAtZero && endedAtZero {
		return time.Time{}, time.Time{}, true, nil
	}
	if startedAtZero || endedAtZero {
		return time.Time{}, time.Time{}, false, fmt.Errorf("started_at and ended_at must be provided together")
	}

	startedAt = startedAt.UTC()
	endedAt = endedAt.UTC()
	if !endedAt.After(startedAt) {
		return time.Time{}, time.Time{}, false, fmt.Errorf("ended_at must be after started_at")
	}

	return startedAt, endedAt, false, nil
}

func nullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	return sql.NullString{String: trimmed, Valid: trimmed != ""}
}

func nullPID(pid int32) sql.NullInt64 {
	if pid <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(pid), Valid: true}
}
