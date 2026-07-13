package reporter

import (
	"context"
	"log/slog"
	"sync"

	"github.com/jonoton/go-ringbuffer"
	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

type PubSubReporter struct {
	mu          sync.Mutex
	subscribers []chan sessionnew.ForegroundProcess
	closed      bool
	current     *sessionnew.ForegroundProcess
}

func From(ctx context.Context, stream *ringbuffer.RingBuffer[sessionnew.ForegroundProcess]) (*PubSubReporter, func()) {
	subscribers := []chan sessionnew.ForegroundProcess{}

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

		for _, ch := range reporter.subscribers {
			close(ch)
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				stream.Stop()
				cleanup()
				return
			case item := <-stream.GetChan():
				reporter.publish(item)
			}
		}
	}()

	return &reporter, cleanup
}

func (p *PubSubReporter) Subscribe() <-chan sessionnew.ForegroundProcess {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	// channel size: there can only be 1 foreground process at a time
	// order: can be processed out of order, we have timestamps
	c := make(chan sessionnew.ForegroundProcess, 1)
	p.subscribers = append(p.subscribers, c)

	return c
}

func (p *PubSubReporter) publish(incoming sessionnew.ForegroundProcess) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	isPublished := isDuplicate(p.current, incoming)
	current := incoming
	p.current = &current
	if isPublished {
		return
	}

	for _, ch := range p.subscribers {
		select {
		case ch <- incoming:
		// monitor: drops old
		// reporter: drops incoming
		// consider: if incoming keeps changing then it's noise
		default:
			slog.Error("dropped") // TODO
		}
	}
}
