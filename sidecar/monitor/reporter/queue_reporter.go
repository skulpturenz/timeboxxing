package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goptics/varmq"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

type TransitionEvent struct {
	ApplicationName string
	Reason          string
	StartedAt       time.Time
	EndedAt         time.Time
	Browser         bool
	Tab             *string
	Idle            bool
	CdpUrl          *string
}

type QueueReporter struct {
	queue varmq.PersistentQueue[any]
}

func NewQueueReporter(queue varmq.PersistentQueue[any]) *QueueReporter {
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

	key := t.From.Key
	appName := strings.TrimSpace(key.AppName)
	event := TransitionEvent{
		ApplicationName: appName,
		Reason:          string(t.Reason),
		StartedAt:       t.From.StartedAt,
		EndedAt:         t.From.EndedAt,
		Browser:         browser.IsBrowser(appName) != browser.BrowserNone,
		Tab:             stringPtr(key.TabTitle),
		Idle:            key.IsIdle,
		CdpUrl:          stringPtr(key.CDPURL),
	}

	if ok := q.queue.Add(event); !ok {
		return fmt.Errorf("add transition event to queue")
	}

	return nil
}
