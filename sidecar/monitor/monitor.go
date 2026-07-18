package monitor

import (
	"context"
	"time"

	"github.com/jonoton/go-ringbuffer"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/idle"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/platform"
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

type MonitorOptions struct {
	PollInterval *time.Duration
	IdleAfter    *time.Duration
	BufferSize   *int
	Reporter     *Reporter
}

type Monitor struct {
	Stream       *ringbuffer.RingBuffer[ForegroundProcess]
	pollInterval time.Duration
	idleAfter    time.Duration
	idleDetector idle.IdleDetector
}

func New(ctx context.Context, options MonitorOptions) *Monitor {
	tracker, err := platform.New(ctx, platform.Config{})
	if err != nil {
		return nil
	}

	idleDetector, err := idle.New(ctx)
	if err != nil {
		idleDetector = idle.Nop()
	}

	pollInterval := 200 * time.Millisecond
	if options.PollInterval != nil {
		pollInterval = *options.PollInterval
	}

	idleAfter := 5 * time.Minute
	if options.IdleAfter != nil {
		idleAfter = *options.IdleAfter
	}

	bufferSize := 10
	if options.BufferSize != nil {
		bufferSize = *options.BufferSize
	}

	// don't care about dropping oldest items
	// if we poll every 200ms, it takes 2 seconds to fill
	buffer := ringbuffer.New[ForegroundProcess](bufferSize)

	monitor := Monitor{
		pollInterval: pollInterval,
		idleAfter:    idleAfter,
		Stream:       buffer,
		idleDetector: idleDetector,
	}

	go monitor.Poll(ctx, tracker)

	return &monitor
}

func (m *Monitor) Poll(ctx context.Context, tracker platform.Tracker) {
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
			Timestamp: time.Now().Add(time.Duration(-1*idleSeconds) * time.Second),
		}
		m.Stream.Add(item)

		return
	}

	appName := windowInfo.AppName
	appIdentifier := windowInfo.AppIdentifier
	appPath := windowInfo.AppPath
	pid := windowInfo.PID
	windowTitle := windowInfo.WindowTitle
	titleSource := windowInfo.TitleSource
	timestamp := windowInfo.Timestamp
	encrichments := map[string]any{}

	item := ForegroundProcess{
		AppName:       &appName,
		AppIdentifier: &appIdentifier,
		AppPath:       &appPath,
		PID:           &pid,
		WindowTitle:   &windowTitle,
		TitleSource:   &titleSource,
		Timestamp:     timestamp,
		Idle:          false,
		Enrichments:   encrichments,
	}

	m.Stream.Add(item)
}
