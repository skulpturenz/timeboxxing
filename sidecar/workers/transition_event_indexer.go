package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goptics/varmq"
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type TransitionEventIndexed struct {
	TransitionEventId int64
}

func (s WorkerServices) TransitionEventIndexerWorker(ctx context.Context, q queue.Queue) (varmq.PersistentQueue[any], func()) {
	logger := s.Logger.With("worker", "transition_event_indexer")

	queue, _, cleanup := q.NewWorker(ctx, func(j varmq.Job[any]) {
		event, err := transitionEventReportedFromJobData(j.Data())
		assert.Nil(err)
		if err != nil {
			logger.ErrorContext(ctx, "invalid transition event reported job type", "type", fmt.Sprintf("%T", j.Data()), "error", err)
			return
		}

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

	return queue, cleanup
}

func transitionEventReportedFromJobData(data any) (TransitionEventReported, error) {
	if event, ok := data.(TransitionEventReported); ok {
		return event, nil
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return TransitionEventReported{}, err
	}

	var event TransitionEventReported
	if err := json.Unmarshal(encoded, &event); err != nil {
		return TransitionEventReported{}, err
	}

	return event, nil
}
