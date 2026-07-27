package transitions

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsoperatingsystem "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_operating_system"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func (s *Service) RecordTransitionEvent(ctx context.Context, params RecordTransitionEventParams) (int64, error) {
	if s == nil || s.writeTx == nil {
		return 0, fmt.Errorf("transition event writer is unavailable")
	}

	startedAt, endedAt, err := transitionEventTimestamps(params.StartedAt, params.EndedAt)
	if err != nil {
		return 0, err
	}

	var timelineID int64
	if err := s.writeTx.WriteTx(ctx, func(q *writequeries.Queries) error {
		appName := strings.TrimSpace(params.ApplicationName)
		var applicationID *int64

		if !params.Idle && appName != "" {
			os, err := enumsoperatingsystem.Parse(runtime.GOOS)
			if err != nil {
				return err
			}
			id, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
				Name:              appName,
				Identifier:        utils.ZeroNil(strings.TrimSpace(params.ApplicationIdentifier)),
				OperatingSystemID: int64(os),
				Path:              utils.ZeroNil(strings.TrimSpace(params.ApplicationPath)),
			})
			if err != nil {
				return fmt.Errorf("upsert application %q: %w", appName, err)
			}
			applicationID = &id
		}

		pid := utils.ZeroNil(int64(params.PID))

		// The event store keeps one foreground_processes row per boundary timestamp. Consecutive
		// sessions share a boundary (EndedAt(N) == StartedAt(N+1)); the created_at_utc upsert makes
		// the end row of one session and the initial row of the next the same physical row.
		initialForegroundProcessID, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
			ApplicationID: applicationID,
			Pid:           pid,
			CreatedAtUtc:  startedAt,
		})
		if err != nil {
			return fmt.Errorf("upsert initial foreground process: %w", err)
		}

		if _, err := q.InsertForegroundProcessMetadata(ctx, writequeries.InsertForegroundProcessMetadataParams{
			ForegroundProcessID: initialForegroundProcessID,
			Browser:             params.Browser,
			Idle:                params.Idle,
			Tab:                 params.Tab,
			CdpUrl:              params.CDPURL,
			Latitude:            params.Latitude,
			Longitude:           params.Longitude,
			PublicIp:            params.PublicIP,
		}); err != nil {
			return fmt.Errorf("insert foreground process metadata: %w", err)
		}

		endForegroundProcessID, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
			ApplicationID: applicationID,
			Pid:           pid,
			CreatedAtUtc:  endedAt,
		})
		if err != nil {
			return fmt.Errorf("upsert end foreground process: %w", err)
		}

		timelineID, err = q.UpsertTimeline(ctx, writequeries.UpsertTimelineParams{
			InitialForegroundProcessID: &initialForegroundProcessID,
			EndForegroundProcessID:     &endForegroundProcessID,
		})
		if err != nil {
			return fmt.Errorf("upsert timeline: %w", err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	if err := s.PublishTransitionEvent(ctx, PublishTransitionEventParams{ID: timelineID}); err != nil {
		return 0, fmt.Errorf("publish transition event: %w", err)
	}

	return timelineID, nil
}

func transitionEventTimestamps(startedAt time.Time, endedAt time.Time) (time.Time, time.Time, error) {
	if startedAt.IsZero() || endedAt.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("started_at and ended_at must be provided")
	}

	startedAt = startedAt.UTC()
	endedAt = endedAt.UTC()
	if endedAt.Before(startedAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("ended_at must not be before started_at")
	}

	return startedAt, endedAt, nil
}
