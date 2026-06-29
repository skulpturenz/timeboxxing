package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type TransitionEvent struct {
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
	PID                   int32
	Reason                string
	StartedAt             time.Time
	EndedAt               time.Time
	Browser               bool
	Tab                   *string
	Idle                  bool
	CdpUrl                *string
}

type TransitionEventQueue interface {
	Add(TransitionEvent) error
}

type TransitionEventIndexer interface {
	IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error)
}

type QueueReporter struct {
	queue TransitionEventQueue
}

type transitionEventQueueKey struct{}

func RegisterTransitionEventQueue(registry *services.Services[any, any], queue TransitionEventQueue) {
	services.Set(registry, transitionEventQueueKey{}, queue)
}

func TransitionEventQueueFromServices(registry *services.Services[any, any]) (TransitionEventQueue, bool) {
	service, ok := services.Get[TransitionEventQueue](registry, transitionEventQueueKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func NewQueueReporter(registry *services.Services[any, any]) *QueueReporter {
	queue, _ := TransitionEventQueueFromServices(registry)
	return &QueueReporter{queue: queue}
}

func (q *QueueReporter) Run(ctx context.Context, transitions <-chan session.Transition) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t, ok := <-transitions:
			if !ok {
				return nil
			}
			if err := q.Record(ctx, t); err != nil {
				return err
			}
		}
	}
}

func (q *QueueReporter) Record(_ context.Context, t session.Transition) error {
	if t.From == nil || t.From.IsOpen() {
		return nil
	}
	if q.queue == nil {
		return fmt.Errorf("transition event queue is unavailable")
	}

	key := t.From.Key
	appName := strings.TrimSpace(key.AppName)
	identity := t.From.ApplicationIdentity
	event := TransitionEvent{
		ApplicationName:       appName,
		ApplicationIdentifier: strings.TrimSpace(identity.Identifier),
		ApplicationPath:       strings.TrimSpace(identity.Path),
		PID:                   identity.PID,
		Reason:                string(t.Reason),
		StartedAt:             t.From.StartedAt,
		EndedAt:               t.From.EndedAt,
		Browser:               browser.IsBrowser(appName) != browser.BrowserNone,
		Tab:                   stringPtr(key.TabTitle),
		Idle:                  key.IsIdle,
		CdpUrl:                stringPtr(key.CDPURL),
	}

	return q.queue.Add(event)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
