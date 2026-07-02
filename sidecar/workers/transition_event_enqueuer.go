package workers

import (
	"context"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type TransitionEventReportedEnqueuer struct {
	queue *queue.Queue[TransitionEventReported]
}

func NewTransitionEventReportedEnqueuer(q *queue.Queue[TransitionEventReported]) *TransitionEventReportedEnqueuer {
	return &TransitionEventReportedEnqueuer{queue: q}
}

func (e *TransitionEventReportedEnqueuer) EnqueueTransitionEvent(_ context.Context, transitionEventID int64) error {
	if e == nil || e.queue == nil {
		return fmt.Errorf("transition event reported queue is unavailable")
	}
	if err := e.queue.Add(TransitionEventReported{TransitionEventId: transitionEventID}); err != nil {
		return fmt.Errorf("enqueue transition event reported job: %w", err)
	}
	return nil
}
