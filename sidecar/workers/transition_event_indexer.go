package workers

import (
	"context"
	"time"

	"github.com/goptics/varmq"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

const transitionEventIndexAttempts = 3

// TransitionEventReported is enqueued when a timeline entry is finalized, to be picked up by the
// semantic indexer worker below.
type TransitionEventReported struct {
	TransitionEventId int64
}

type TransitionEventIndexed struct {
	TransitionEventId int64
}

func (r *Runtime) TransitionEventIndexerWorker(ctx context.Context, q *queue.Queue[TransitionEventReported]) func() {
	logger := r.logger.With("worker", "transition_event_indexer")

	cleanup := q.AddWorker(ctx, func(j varmq.Job[TransitionEventReported]) {
		event := j.Data()
		assert.NotNil(r.indexer)
		if r.indexer == nil {
			logger.ErrorContext(ctx, "transition event indexer is nil", "transition_event_id", event.TransitionEventId)
			return
		}

		var (
			documentID int64
			err        error
		)
		for attempt := 1; attempt <= transitionEventIndexAttempts; attempt++ {
			documentID, err = r.indexer.IndexTransitionEvent(ctx, event.TransitionEventId)
			if err == nil {
				break
			}
			if attempt < transitionEventIndexAttempts {
				time.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
			}
		}
		if err != nil {
			logger.ErrorContext(ctx, "failed to index transition event", "transition_event_id", event.TransitionEventId, "error", err)
			return
		}

		// s.Queues.TransitionEventIndexedQueue.Add(TransitionEventIndexed{
		// 	TransitionEventId: event.TransitionEventId,
		// })

		logger.InfoContext(ctx, "indexed transition event", "transition_event_id", event.TransitionEventId, "transition_event_document_id", documentID)
	}, 0)

	return cleanup
}
