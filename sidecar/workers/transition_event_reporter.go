package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func (s WorkerServices) TransitionEventReporterWorker(ctx context.Context, q queue.Queue) (varmq.PersistentQueue[any], func()) {
	logger := s.Logger.With("worker", "transition_event")

	queue, _, cleanup := q.NewWorker(ctx, func(j varmq.Job[any]) {
		event, err := transitionEventFromJobData(j.Data())
		assert.Nil(err)
		if err != nil {
			logger.ErrorContext(ctx, "invalid transition event job type", "type", fmt.Sprintf("%T", j.Data()), "error", err)
			return
		}

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

		s.Queues.TransitionEventReportedQueue.Add(TransitionEventReported{
			TransitionEventId: eventID,
		})

		logger.InfoContext(ctx, "recorded transition event", "transition_event_id", eventID)
	}, 0)

	return queue, cleanup
}

func transitionEventFromJobData(data any) (reporter.TransitionEvent, error) {
	if event, ok := data.(reporter.TransitionEvent); ok {
		return event, nil
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return reporter.TransitionEvent{}, err
	}

	var event reporter.TransitionEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		return reporter.TransitionEvent{}, err
	}

	return event, nil
}
