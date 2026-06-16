package workers

import (
	"context"
	"database/sql"
	"strings"

	"github.com/goptics/varmq"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type TransitionEventReported struct {
	TransitionEventId int64
}

func (s WorkerServices) TransitionEventReporterWorker(ctx context.Context, q *queue.Queue[reporter.TransitionEvent]) func() {
	logger := s.Logger.With("worker", "transition_event")

	cleanup := q.AddWorker(ctx, func(j varmq.Job[reporter.TransitionEvent]) {
		event := j.Data()
		assert.NotNil(s.WriteConn)

		tx, err := s.WriteConn.BeginTx(ctx, nil)
		assert.Nil(err)

		if err != nil {
			logger.ErrorContext(ctx, "begin transition event transaction", "error", err)
			return
		}
		defer tx.Rollback()

		q := queries.New(tx)
		appName := strings.TrimSpace(event.ApplicationName)
		applicationID := sql.NullInt64{}

		if !event.Idle && appName != "" {
			id, err := q.UpsertApplication(ctx, appName)
			if err != nil {
				logger.ErrorContext(ctx, "upsert application", "application", appName, "error", err)
				return
			}

			applicationID = sql.NullInt64{Int64: id, Valid: true}
		}

		eventID, err := q.CreateTransitionEvent(ctx, queries.CreateTransitionEventParams{
			ApplicationID: applicationID,
			Reason:        event.Reason,
			StartedAt:     event.StartedAt,
			EndedAt:       event.EndedAt,
		})
		if err != nil {
			logger.ErrorContext(ctx, "create transition event", "error", err)
			return
		}

		if err := q.CreateTransitionEventMetadata(ctx, queries.CreateTransitionEventMetadataParams{
			TransitionEventID: eventID,
			Browser:           event.Browser,
			Tab:               event.Tab,
			Idle:              event.Idle,
			CdpUrl:            event.CdpUrl,
		}); err != nil {
			logger.ErrorContext(ctx, "create transition event metadata", "error", err)
			return
		}

		if err := tx.Commit(); err != nil {
			logger.ErrorContext(ctx, "commit transition event transaction", "error", err)
			return
		}

		if err := s.Queues.TransitionEventReportedQueue.Add(TransitionEventReported{
			TransitionEventId: eventID,
		}); err != nil {
			logger.ErrorContext(ctx, "add transition event reported job", "transition_event_id", eventID, "error", err)
			return
		}

		logger.InfoContext(ctx, "recorded transition event", "transition_event_id", eventID)
	}, 0)

	return cleanup
}
