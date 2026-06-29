package workers

import (
	"context"
	"fmt"

	"github.com/goptics/varmq"
	"github.com/negrel/assert"
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type TransitionEventReported struct {
	TransitionEventId int64
}

func (r *Runtime) TransitionEventReporterWorker(ctx context.Context, q *queue.Queue[reporter.TransitionEvent]) func() {
	logger := r.logger.With("worker", "transition_event")

	cleanup := q.AddWorker(ctx, func(j varmq.Job[reporter.TransitionEvent]) {
		event := j.Data()
		assert.NotNil(r.transitions)
		eventID, err := r.transitions.RecordTransitionEvent(ctx, componentTransitions.RecordTransitionEventParams{
			ApplicationName:       event.ApplicationName,
			ApplicationIdentifier: event.ApplicationIdentifier,
			ApplicationPath:       event.ApplicationPath,
			PID:                   event.PID,
			Reason:                event.Reason,
			StartedAt:             event.StartedAt,
			EndedAt:               event.EndedAt,
			Browser:               event.Browser,
			Tab:                   event.Tab,
			Idle:                  event.Idle,
			CDPURL:                event.CdpUrl,
		})
		if err != nil {
			logger.ErrorContext(ctx, "record transition event", "error", err)
			return
		}

		if r.indexer != nil {
			if r.queues.TransitionEventReportedQueue == nil {
				logger.ErrorContext(ctx, "transition event reported queue unavailable", "transition_event_id", eventID, "error", fmt.Errorf("queue is nil"))
				return
			}
			if err := r.queues.TransitionEventReportedQueue.Add(TransitionEventReported{
				TransitionEventId: eventID,
			}); err != nil {
				logger.ErrorContext(ctx, "add transition event reported job", "transition_event_id", eventID, "error", err)
				return
			}
		}

		logger.InfoContext(ctx, "recorded transition event", "transition_event_id", eventID)
	}, 0)

	return cleanup
}
