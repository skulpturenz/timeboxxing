package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/idle"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/platform"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

type Config struct {
	PollInterval  time.Duration
	MinDuration   time.Duration
	IdleThreshold time.Duration
}

type Handle struct {
	Transitions <-chan session.Transition
	manager     *session.SessionManager
}

func (h *Handle) CurrentSession() *session.Session {
	if h == nil || h.manager == nil {
		return nil
	}
	return h.manager.CurrentSession()
}

func Start(ctx context.Context, logger *slog.Logger, cfg Config) (*Handle, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.MinDuration == 0 {
		cfg.MinDuration = time.Second
	}
	if cfg.IdleThreshold == 0 {
		cfg.IdleThreshold = 5 * time.Minute
	}

	tracker, err := platform.New(ctx, platform.Config{})
	if err != nil {
		return nil, fmt.Errorf("create platform tracker: %w", err)
	}

	idleDetector, err := idle.New(ctx)
	if err != nil {
		logger.WarnContext(ctx, "idle detection unavailable, continuing without idle tracking", "error", err)
		idleDetector = idle.Nop()
	}

	manager := session.NewManager(session.ManagerConfig{
		MinDuration:   cfg.MinDuration,
		IdleThreshold: cfg.IdleThreshold,
		IdleDetector:  idleDetector,
	})

	go manager.Run(ctx, tracker, cfg.PollInterval)

	return &Handle{
		Transitions: manager.Transitions,
		manager:     manager,
	}, nil
}
