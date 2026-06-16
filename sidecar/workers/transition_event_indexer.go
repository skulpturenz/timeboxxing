package workers

import (
	"context"

	"github.com/goptics/varmq"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type TransitionEventIndexed struct {
	TransitionEventId int64
}

func (s WorkerServices) TransitionEventIndexerWorker(ctx context.Context, q *queue.Queue[TransitionEventReported]) func() {
	logger := s.Logger.With("worker", "transition_event_indexer")

	cleanup := q.AddWorker(ctx, func(j varmq.Job[TransitionEventReported]) {
		event := j.Data()
		assert.NotNil(s.TransitionEventIndexer)
		if s.TransitionEventIndexer == nil {
			logger.ErrorContext(ctx, "transition event indexer is nil", "transition_event_id", event.TransitionEventId)
			return
		}

		documentID, err := s.TransitionEventIndexer.IndexTransitionEvent(ctx, event.TransitionEventId)
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
