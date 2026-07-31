package workers

import (
	"context"
	"fmt"
)

type TransitionEventReportedEnqueuer struct {
	ch chan<- TransitionEventReported
}

func NewTransitionEventReportedEnqueuer(ch chan<- TransitionEventReported) *TransitionEventReportedEnqueuer {
	return &TransitionEventReportedEnqueuer{ch: ch}
}

func (e *TransitionEventReportedEnqueuer) EnqueueTransitionEvent(ctx context.Context, transitionEventID int64) error {
	if e == nil || e.ch == nil {
		return fmt.Errorf("transition event reported queue is unavailable")
	}

	select {
	case e.ch <- TransitionEventReported{TransitionEventId: transitionEventID}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("enqueue transition event reported job: %w", ctx.Err())
	}
}
