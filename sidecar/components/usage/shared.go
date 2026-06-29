package usage

import (
	"strings"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

type Source int

const (
	SourceApplication Source = iota + 1
	SourceBrowser
	SourceIdle
)

type Event struct {
	ID                    int64
	Title                 string
	SourceName            string
	Source                Source
	StartedAt             time.Time
	EndedAt               time.Time
	Reason                string
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
	PID                   int32
	CDPURL                string
	Active                bool
}

type Window struct {
	StartedAt time.Time
	EndedAt   time.Time
}

type GetEventsParams struct {
	Window Window
}

type SubscribeParams struct {
	Window Window
}

type Subscription struct {
	Events <-chan Event
	Close  func()
}

func eventOverlapsWindow(event componentTransitions.Event, window Window) bool {
	return event.EndedAt.After(window.StartedAt) && event.StartedAt.Before(window.EndedAt)
}

func usageEventOverlapsWindow(event Event, window Window) bool {
	return event.EndedAt.After(window.StartedAt) && event.StartedAt.Before(window.EndedAt)
}

func eventFromTransition(event componentTransitions.Event) Event {
	applicationName := strings.TrimSpace(event.ApplicationName)
	tab := strings.TrimSpace(event.Tab)

	usageEvent := Event{
		ID:              event.ID,
		StartedAt:       event.StartedAt,
		EndedAt:         event.EndedAt,
		Reason:          event.Reason,
		ApplicationName: applicationName,
		PID:             event.PID,
		CDPURL:          event.CDPURL,
	}
	if !event.Idle {
		usageEvent.ApplicationIdentifier = strings.TrimSpace(event.ApplicationIdentifier)
		usageEvent.ApplicationPath = strings.TrimSpace(event.ApplicationPath)
	}

	switch {
	case event.Idle:
		usageEvent.Title = "Idle"
		usageEvent.SourceName = "Idle"
		usageEvent.Source = SourceIdle
	case event.Browser:
		usageEvent.Title = firstNonEmpty(tab, applicationName, "Browser")
		usageEvent.SourceName = firstNonEmpty(applicationName, "Browser")
		usageEvent.Source = SourceBrowser
	default:
		usageEvent.Title = firstNonEmpty(applicationName, "Application")
		usageEvent.SourceName = firstNonEmpty(applicationName, "Application")
		usageEvent.Source = SourceApplication
	}

	return usageEvent
}

func eventFromActiveSession(activeSession *session.Session, now time.Time, window Window) (Event, bool) {
	if activeSession == nil || activeSession.StartedAt.IsZero() {
		return Event{}, false
	}
	endedAt := minTime(now, window.EndedAt)
	if !endedAt.After(activeSession.StartedAt) {
		return Event{}, false
	}

	key := activeSession.Key
	event := eventFromTransition(componentTransitions.Event{
		ID:                    -1,
		ApplicationName:       key.AppName,
		ApplicationIdentifier: activeSession.ApplicationIdentity.Identifier,
		ApplicationPath:       activeSession.ApplicationIdentity.Path,
		PID:                   activeSession.ApplicationIdentity.PID,
		Reason:                "active",
		StartedAt:             activeSession.StartedAt,
		EndedAt:               endedAt,
		Browser:               !key.IsIdle && browser.IsBrowser(key.AppName) != browser.BrowserNone,
		Tab:                   key.TabTitle,
		Idle:                  key.IsIdle,
		CDPURL:                key.CDPURL,
	})
	event.Active = true
	if !usageEventOverlapsWindow(event, window) {
		return Event{}, false
	}

	return event, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func minTime(first time.Time, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}
