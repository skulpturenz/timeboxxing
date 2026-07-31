package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/negrel/assert"
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

// TransitionEventIndexerWorker consumes reported transition events delivered by the persistent
// queue and semantically indexes each one. It returns a cleanup that blocks until the consumer
// goroutine has stopped — on ctx cancellation or the delivery channel closing.
func (r *Runtime) TransitionEventIndexerWorker(ctx context.Context, events <-chan TransitionEventReported) func() {
	logger := r.logger.With("worker", "transition_event_indexer")
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				r.indexTransitionEvent(ctx, logger, event)
			}
		}
	}()

	return func() { <-done }
}

func (r *Runtime) indexTransitionEvent(ctx context.Context, logger *slog.Logger, event TransitionEventReported) {
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

	logger.InfoContext(ctx, "indexed transition event", "transition_event_id", event.TransitionEventId, "transition_event_document_id", documentID)
}
