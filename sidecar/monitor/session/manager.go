package session

import (
	"context"
	"sync"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/idle"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/platform"
)

// TransitionReason describes why the active session changed.
type TransitionReason string

const (
	ReasonFocusChange TransitionReason = "focus_change"
	ReasonIdle        TransitionReason = "idle"
	ReasonReturn      TransitionReason = "return_from_idle"
	ReasonTabChange   TransitionReason = "tab_change"
	ReasonShutdown    TransitionReason = "shutdown"
	ReasonStart       TransitionReason = "start"
)

// Transition is emitted on the Transitions channel whenever the active session changes.
type Transition struct {
	From   *Session // nil on the very first session
	To     *Session // nil on shutdown (final flush)
	Reason TransitionReason
}

// ManagerConfig holds tunable parameters for SessionManager.
type ManagerConfig struct {
	MinDuration   time.Duration // sessions shorter than this are discarded (default 1s)
	IdleThreshold time.Duration // seconds without input → idle (default 5m)
	NoBrowserTabs bool          // collapse all browser tab keys to app name only
	IdleDetector  idle.IdleDetector
	CDPPoller     *browser.CDPPoller // nil → no URL enrichment
}

// SessionManager is the core state machine that tracks session transitions.
type SessionManager struct {
	cfg         ManagerConfig
	mu          sync.Mutex
	current     *Session
	history     []*Session
	stats       map[AppKey]time.Duration
	Transitions chan Transition // buffered; read by the reporter goroutine
}

// NewManager creates a SessionManager with the provided config.
func NewManager(cfg ManagerConfig) *SessionManager {
	// 0 means "no minimum" — callers (e.g. main.go) set their own defaults via flags.
	if cfg.IdleThreshold == 0 {
		cfg.IdleThreshold = 5 * time.Minute
	}
	if cfg.IdleDetector == nil {
		cfg.IdleDetector = idle.Nop()
	}
	return &SessionManager{
		cfg:         cfg,
		stats:       make(map[AppKey]time.Duration),
		Transitions: make(chan Transition, 64),
	}
}

// Run starts the poll loop. It blocks until ctx is cancelled.
func (m *SessionManager) Run(ctx context.Context, tracker platform.Tracker, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.shutdown()
			return
		case <-ticker.C:
			m.tick(ctx, tracker)
		}
	}
}

// History returns a snapshot of all closed sessions.
func (m *SessionManager) History() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, len(m.history))
	copy(out, m.history)
	return out
}

// Stats returns a copy of accumulated time per AppKey (closed sessions only).
func (m *SessionManager) Stats() map[AppKey]time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[AppKey]time.Duration, len(m.stats))
	for k, v := range m.stats {
		out[k] = v
	}
	return out
}

// CurrentSession returns a copy of the currently open session (may be nil).
func (m *SessionManager) CurrentSession() *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	c := *m.current
	return &c
}

// tick polls for the active window and drives session transitions.
func (m *SessionManager) tick(ctx context.Context, tracker platform.Tracker) {
	info, err := tracker.Poll(ctx)
	if err != nil || info.AppName == "" {
		return
	}

	idleSecs, _ := m.cfg.IdleDetector.SecondsSinceLastInput(ctx)
	isIdle := idleSecs >= m.cfg.IdleThreshold.Seconds()

	var newKey AppKey
	if isIdle {
		newKey = AppKey{IsIdle: true}
	} else {
		newKey = m.buildKey(ctx, info)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		// Very first observation.
		s := &Session{Key: newKey, StartedAt: info.Timestamp}
		m.current = s
		m.emit(Transition{From: nil, To: s, Reason: ReasonStart})
		return
	}

	if newKey == m.current.Key {
		// No change — handle the edge case where AppName is the same but
		// WindowTitle is now non-empty (app finishing load).
		return
	}

	// Edge case: same app, new window title is empty while app is loading → ignore.
	if !isIdle &&
		info.AppName == m.current.Key.AppName &&
		info.WindowTitle == "" &&
		m.current.Key.TabTitle == "" {
		return
	}

	m.transition(newKey, info.Timestamp)
}

// transition closes the current session and opens a new one.
// Must be called with m.mu held.
func (m *SessionManager) transition(newKey AppKey, at time.Time) {
	old := m.current
	elapsed := at.Sub(old.StartedAt)

	reason := transitionReason(old.Key, newKey)

	if elapsed >= m.cfg.MinDuration {
		old.Close(at)
		m.history = append(m.history, old)
		m.stats[old.Key] += old.Duration
		newSess := &Session{Key: newKey, StartedAt: at}
		m.current = newSess
		m.emit(Transition{From: old, To: newSess, Reason: reason})
	} else {
		// Sub-threshold: discard the flash session silently.
		// Preserve the original StartedAt so accumulated time is correct.
		origStart := old.StartedAt
		newSess := &Session{Key: newKey, StartedAt: origStart}
		m.current = newSess
	}
}

// shutdown closes the current open session and closes the Transitions channel.
func (m *SessionManager) shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil && m.current.IsOpen() {
		now := time.Now()
		m.current.Close(now)
		m.history = append(m.history, m.current)
		m.stats[m.current.Key] += m.current.Duration
		m.emit(Transition{From: m.current, To: nil, Reason: ReasonShutdown})
	}
	close(m.Transitions)
}

func (m *SessionManager) emit(t Transition) {
	select {
	case m.Transitions <- t:
	default:
		// Channel full — reporter is lagging. Drop to avoid blocking the poll loop.
	}
}

// buildKey converts a WindowInfo into an AppKey, handling browser tab extraction.
func (m *SessionManager) buildKey(ctx context.Context, info platform.WindowInfo) AppKey {
	if m.cfg.NoBrowserTabs {
		return AppKey{AppName: info.AppName}
	}
	kind := browser.IsBrowser(info.AppName)
	if kind != browser.BrowserNone && info.WindowTitle != "" {
		tab, ok := browser.ParseTabTitle(info.AppName, info.WindowTitle)
		if ok {
			url := ""
			if m.cfg.CDPPoller != nil {
				url = m.cfg.CDPPoller.URLForTitle(ctx, tab.TabTitle)
			}
			return AppKey{AppName: info.AppName, TabTitle: tab.TabTitle, CDPURL: url}
		}
	}
	return AppKey{AppName: info.AppName}
}

func transitionReason(from, to AppKey) TransitionReason {
	switch {
	case to.IsIdle:
		return ReasonIdle
	case from.IsIdle:
		return ReasonReturn
	case from.AppName == to.AppName && from.TabTitle != to.TabTitle:
		return ReasonTabChange
	default:
		return ReasonFocusChange
	}
}
