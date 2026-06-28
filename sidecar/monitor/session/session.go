package session

import (
	"fmt"
	"time"
)

// AppKey uniquely identifies a trackable entity.
// For browser tabs, TabTitle distinguishes between tabs in the same browser.
// For all other apps, only AppName is set.
type AppKey struct {
	AppName  string
	TabTitle string // non-empty only for browser windows
	IsIdle   bool   // synthetic idle session
	CDPURL   string // URL from Chrome DevTools Protocol, if available
}

type AppIdentity struct {
	Identifier string
	Path       string
}

func (k AppKey) DisplayName() string {
	if k.IsIdle {
		return "[Idle]"
	}
	if k.TabTitle != "" {
		return fmt.Sprintf("%s | %s", k.AppName, k.TabTitle)
	}
	return k.AppName
}

// Session is a closed (or open) time interval of continuous focus on one AppKey.
type Session struct {
	Key                 AppKey
	ApplicationIdentity AppIdentity
	StartedAt           time.Time
	EndedAt             time.Time     // zero while the session is still open
	Duration            time.Duration // computed when the session is closed
}

func (s *Session) IsOpen() bool   { return s.EndedAt.IsZero() }
func (s *Session) IsClosed() bool { return !s.EndedAt.IsZero() }

// Close finalises an open session at the given time.
func (s *Session) Close(at time.Time) {
	s.EndedAt = at
	s.Duration = at.Sub(s.StartedAt)
}
