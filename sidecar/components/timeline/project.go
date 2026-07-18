package timeline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"strings"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsoperatingsystem "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_operating_system"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/location"
)

// Project persists one enriched foreground-process observation and reconciles it against the latest
// timeline entry. The reconciliation key is (pid, idle, tab) — exactly what is persisted in
// foreground_process_metadata — so consecutive samples that share it extend the open entry, browser
// tab switches (distinct tab) start new entries, and non-browser window-title changes (tab stays nil)
// collapse into one entry.
func (s *Service) Project(ctx context.Context, fp monitor.ForegroundProcess) error {
	if s == nil || s.writeTx == nil || s.readQuerier == nil {
		return fmt.Errorf("timeline projector is unavailable")
	}

	// Read the latest timeline entry on the read connection BEFORE the write tx. This is race-free
	// because the projector is the sole writer of timeline/foreground_processes and runs single-
	// goroutine (draining one reporter subscription), so this reflects all prior committed entries.
	latest, err := s.readQuerier.GetLatestTimelineEntry(ctx)
	hasLatest := true
	if errors.Is(err, sql.ErrNoRows) {
		hasLatest = false
	} else if err != nil {
		return fmt.Errorf("get latest timeline entry: %w", err)
	}

	name := strings.TrimSpace(deref(fp.AppName))
	identifier := deref(fp.AppIdentifier)
	path := deref(fp.AppPath)
	pid := int64(deref(fp.PID))
	idle := fp.Idle
	observedAt := fp.Timestamp.UTC()
	if observedAt.IsZero() {
		observedAt = s.clock().UTC()
	}

	isBrowser := !idle && browser.IsBrowser(name) != browser.BrowserNone
	var tab, cdpURL *string
	if resolved, ok := browser.Get(fp); ok {
		if title := strings.TrimSpace(resolved.Title); title != "" {
			tab = &resolved.Title
		}
		if url := strings.TrimSpace(resolved.URL); url != "" {
			cdpURL = &resolved.URL
		}
	}
	var latitude, longitude *float64
	var publicIP *string
	if env, ok := location.Get(fp); ok {
		latitude, longitude, publicIP = env.Latitude, env.Longitude, env.PublicIP
	}

	var finalizedID int64
	if err := s.writeTx.WriteTx(ctx, func(q *writequeries.Queries) error {
		finalizedID = 0

		applicationID := sql.NullInt64{}
		if !idle && name != "" {
			id, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
				Name:              name,
				Identifier:        nullString(identifier),
				OperatingSystemID: operatingSystemID(),
				Path:              nullString(path),
			})
			if err != nil {
				return fmt.Errorf("upsert application %q: %w", name, err)
			}
			applicationID = sql.NullInt64{Int64: id, Valid: true}
		}

		// The event store retains every observation so the timeline can be reconstructed after data
		// loss. The upsert makes an end boundary shared with the next entry's initial boundary.
		sampleFpID, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
			ApplicationID: applicationID,
			Pid:           pid,
			CreatedAtUtc:  observedAt,
		})
		if err != nil {
			return fmt.Errorf("upsert foreground process: %w", err)
		}

		if err := q.CreateForegroundProcessMetadata(ctx, writequeries.CreateForegroundProcessMetadataParams{
			ForegroundProcessID: sampleFpID,
			Browser:             isBrowser,
			Idle:                idle,
			Tab:                 tab,
			CdpUrl:              cdpURL,
			Latitude:            latitude,
			Longitude:           longitude,
			PublicIp:            publicIP,
		}); err != nil {
			return fmt.Errorf("create foreground process metadata: %w", err)
		}

		// No prior entry, or the latest is already closed (no open/current entry) — open a new one.
		if !hasLatest || latest.EndForegroundProcessID.Valid {
			return openTimeline(ctx, q, sampleFpID)
		}

		if sameKey(latest, pid, idle, tab) {
			// Still the current context: the entry stays open (end_foreground_process_id NULL); its
			// live duration is now - start, computed at read time. The raw sample is already retained
			// in the event store above for reconstruction.
			return nil
		}

		if observedAt.Sub(latest.InitialCreatedAt.UTC()) >= s.granularity {
			// Long-enough entry: close it at the shared boundary and open a new entry.
			if err := updateTimelineEnd(ctx, q, latest.TimelineID, sampleFpID); err != nil {
				return err
			}
			finalizedID = latest.TimelineID
		} else {
			// Flicker shorter than the granularity: drop it entirely.
			if err := q.DeleteTimeline(ctx, latest.TimelineID); err != nil {
				return fmt.Errorf("delete flicker timeline entry: %w", err)
			}
		}
		return openTimeline(ctx, q, sampleFpID)
	}); err != nil {
		return err
	}

	if finalizedID == 0 {
		return nil
	}

	if s.publisher != nil {
		if err := s.publisher.PublishTransitionEvent(ctx, componentTransitions.PublishTransitionEventParams{ID: finalizedID}); err != nil {
			return fmt.Errorf("publish transition event: %w", err)
		}
	}
	if s.enqueuer != nil {
		if err := s.enqueuer.EnqueueTransitionEvent(ctx, finalizedID); err != nil {
			return fmt.Errorf("enqueue transition event: %w", err)
		}
	}
	return nil
}

func openTimeline(ctx context.Context, q *writequeries.Queries, foregroundProcessID int64) error {
	if _, err := q.CreateTimeline(ctx, writequeries.CreateTimelineParams{
		InitialForegroundProcessID: sql.NullInt64{Int64: foregroundProcessID, Valid: true},
	}); err != nil {
		return fmt.Errorf("create timeline entry: %w", err)
	}
	return nil
}

func updateTimelineEnd(ctx context.Context, q *writequeries.Queries, timelineID int64, foregroundProcessID int64) error {
	if err := q.UpdateTimelineEnd(ctx, writequeries.UpdateTimelineEndParams{
		EndForegroundProcessID: sql.NullInt64{Int64: foregroundProcessID, Valid: true},
		ID:                     timelineID,
	}); err != nil {
		return fmt.Errorf("update timeline end: %w", err)
	}
	return nil
}

// sameKey reports whether the incoming observation belongs to the same timeline entry as latest,
// using the persisted (pid, idle, tab) key.
func sameKey(latest readqueries.GetLatestTimelineEntryRow, pid int64, idle bool, tab *string) bool {
	if latest.InitialPid != pid {
		return false
	}
	if (latest.InitialIdle.Valid && latest.InitialIdle.Bool) != idle {
		return false
	}
	return equalStringPtr(latest.InitialTab, tab)
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// operatingSystemID maps the running platform to a seeded operating_systems id. Unknown platforms
// (e.g. linux, which is not seeded) leave the application's operating_system_id NULL.
func operatingSystemID() sql.NullInt64 {
	switch runtime.GOOS {
	case "darwin":
		return sql.NullInt64{Int64: int64(enumsoperatingsystem.MacOS), Valid: true}
	case "windows":
		return sql.NullInt64{Int64: int64(enumsoperatingsystem.Windows), Valid: true}
	default:
		return sql.NullInt64{}
	}
}

func nullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	return sql.NullString{String: trimmed, Valid: trimmed != ""}
}
