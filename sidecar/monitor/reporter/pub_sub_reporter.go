package reporter

import (
	"context"
	"log/slog"
	"sync"

	"github.com/jonoton/go-ringbuffer"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

type PubSubReporter struct {
	mu          sync.Mutex
	subscribers []*subscriber
	closed      bool
}

type subscriber struct {
	id        string
	ch        chan<- monitor.ForegroundProcess
	current   *monitor.ForegroundProcess
	dropCount int
}

func From(ctx context.Context, stream *ringbuffer.RingBuffer[monitor.ForegroundProcess]) (*PubSubReporter, func()) {
	subscribers := []*subscriber{}

	reporter := PubSubReporter{
		subscribers: subscribers,
	}

	cleanup := func() {
		reporter.mu.Lock()
		defer reporter.mu.Unlock()

		if reporter.closed {
			return
		}

		reporter.closed = true

		for _, s := range reporter.subscribers {
			close(s.ch)
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				cleanup()
				return
			case item := <-stream.GetChan():
				reporter.publish(item)
			}
		}
	}()

	return &reporter, cleanup
}

func (p *PubSubReporter) Subscribe(id string) <-chan monitor.ForegroundProcess {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	// channel size: there can only be 1 foreground process at a time
	// order: can be processed out of order, we have timestamps
	c := make(chan monitor.ForegroundProcess, 1)
	p.subscribers = append(p.subscribers, &subscriber{
		id: id,
		ch: c,
	})

	return c
}

func (p *PubSubReporter) publish(incoming monitor.ForegroundProcess) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	snapshot := new(incoming)
	for _, s := range p.subscribers {
		shouldSkip := isReported(s.current, incoming)
		s.current = snapshot
		if shouldSkip {
			continue
		}

		select {
		case s.ch <- incoming:
		// monitor: drops old
		// reporter: drops incoming
		// consider: if incoming keeps changing then it's noise
		default:
			s.dropCount += 1
			slog.Error("dropped foreground process")
		}
	}
}
