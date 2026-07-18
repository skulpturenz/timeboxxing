package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/jonoton/go-ringbuffer"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/idle"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/platform"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type ForegroundProcess struct {
	AppName       *string
	AppIdentifier *string
	AppPath       *string
	PID           *int32
	WindowTitle   *string
	TitleSource   *platform.TitleSource
	Timestamp     time.Time
	Idle          bool
	Enrichments   map[string]any
}

type Options struct {
	PollInterval *time.Duration
	IdleAfter    *time.Duration
	BufferSize   *int
	Permissions  []permission.Permission
}

type Monitor struct {
	Stream       *ringbuffer.RingBuffer[ForegroundProcess]
	pollInterval time.Duration
	idleAfter    time.Duration
	idleDetector idle.IdleDetector
}

func New(ctx context.Context, options Options) (*Monitor, error) {
	tracker, err := platform.New(ctx, platform.Config{PromptPermissions: true})
	if err != nil {
		return nil, fmt.Errorf("create platform tracker: %w", err)
	}

	idleDetector, err := idle.New(ctx)
	if err != nil {
		idleDetector = idle.Nop()
	}

	for _, perm := range options.Permissions {
		perm.Request(ctx)
	}

	pollInterval := utils.Coalesce(options.PollInterval, 200*time.Millisecond)
	idleAfter := utils.Coalesce(options.IdleAfter, 5*time.Minute)
	bufferSize := utils.Coalesce(options.BufferSize, 10)

	// don't care about dropping oldest items
	// if we poll every 200ms, it takes 2 seconds to fill
	buffer := ringbuffer.New[ForegroundProcess](bufferSize)

	monitor := Monitor{
		pollInterval: pollInterval,
		idleAfter:    idleAfter,
		Stream:       buffer,
		idleDetector: idleDetector,
	}

	go monitor.poll(ctx, tracker)

	return &monitor, nil
}

func (m *Monitor) poll(ctx context.Context, tracker platform.Tracker) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.Stream.Stop()
			return
		case <-ticker.C:
			m.tick(ctx, tracker)
		}
	}
}

func (m *Monitor) tick(ctx context.Context, tracker platform.Tracker) {
	windowInfo, err := tracker.Poll(ctx)
	if err != nil {
		return
	}

	idleSeconds, _ := m.idleDetector.SecondsSinceLastInput(ctx)
	isIdle := idleSeconds >= m.idleAfter.Seconds()

	if isIdle {
		item := ForegroundProcess{
			Idle:      isIdle,
			Timestamp: time.Now().Add(-time.Duration(idleSeconds * float64(time.Second))),
		}
		m.Stream.Add(item)

		return
	}

	item := ForegroundProcess{
		AppName:       new(windowInfo.AppName),
		AppIdentifier: new(windowInfo.AppIdentifier),
		AppPath:       new(windowInfo.AppPath),
		PID:           new(windowInfo.PID),
		WindowTitle:   new(windowInfo.WindowTitle),
		TitleSource:   new(windowInfo.TitleSource),
		Timestamp:     windowInfo.Timestamp,
		Enrichments:   map[string]any{},
	}

	m.Stream.Add(item)
}
