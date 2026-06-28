package usage

import (
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

const defaultActiveSnapshotInterval = 30 * time.Second

type ActiveSessionProvider interface {
	CurrentSession() *session.Session
}

type Service struct {
	transitions            *componentTransitions.Service
	activeSessions         ActiveSessionProvider
	clock                  func() time.Time
	activeSnapshotInterval time.Duration
}

type NewServiceParams struct {
	Transitions            *componentTransitions.Service
	ActiveSessions         ActiveSessionProvider
	Clock                  func() time.Time
	ActiveSnapshotInterval time.Duration
}

func NewService(params NewServiceParams) *Service {
	clock := params.Clock
	if clock == nil {
		clock = time.Now
	}
	interval := params.ActiveSnapshotInterval
	if interval <= 0 {
		interval = defaultActiveSnapshotInterval
	}
	return &Service{
		transitions:            params.Transitions,
		activeSessions:         params.ActiveSessions,
		clock:                  clock,
		activeSnapshotInterval: interval,
	}
}

func minDuration(first time.Duration, second time.Duration) time.Duration {
	if first < second {
		return first
	}
	return second
}
