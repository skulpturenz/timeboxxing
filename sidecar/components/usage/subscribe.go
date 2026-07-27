package usage

import (
	"context"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
)

func (s *Service) Subscribe(ctx context.Context, params SubscribeParams) Subscription {
	ctx, cancel := context.WithCancel(ctx)
	if s == nil || s.transitions == nil {
		events := make(chan Event)
		close(events)
		return Subscription{
			Events: events,
			Close:  cancel,
		}
	}

	transitionSubscription := s.transitions.Subscribe(ctx, componentTransitions.SubscribeParams{})
	events := make(chan Event, 64)

	go func() {
		defer close(events)
		defer transitionSubscription.Close()
		ticker := time.NewTicker(minDuration(s.activeSnapshotInterval, time.Second))
		defer ticker.Stop()

		// Completed sessions are deduplicated by their stable id — the timeline entry, keyed by its
		// unique initial_foreground_process_id — so a session is streamed at most once regardless of
		// how it is discovered (a published transition event or a DB poll). The active (open/current)
		// event is intentionally NOT deduplicated: it is re-sent on each snapshot so its running
		// duration keeps advancing.
		seen := map[int64]struct{}{}

		send := func(event Event) bool {
			select {
			case events <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}

		emitCompleted := func(event Event) bool {
			if _, ok := seen[event.ID]; ok {
				return true
			}
			seen[event.ID] = struct{}{}
			return send(event)
		}

		// pollCompleted picks up sessions written directly to the event store. The production ingest
		// records timeline entries but does not publish transition events, so polling is the only way
		// the live stream learns about newly-completed sessions. Only not-yet-seen completed events
		// are emitted; the active event is handled separately by sendActive.
		pollCompleted := func() bool {
			usageEvents, err := s.GetEvents(ctx, GetEventsParams{Window: params.Window})
			if err != nil {
				return true
			}
			for _, event := range usageEvents {
				if event.Active {
					continue
				}
				if !emitCompleted(event) {
					return false
				}
			}
			return true
		}

		sendActive := func() (keepGoing bool, sent bool) {
			event, ok := s.activeEvent(ctx, params.Window)
			if !ok {
				return true, false
			}
			return send(event), true
		}

		if !pollCompleted() {
			return
		}
		if keepGoing, sent := sendActive(); !keepGoing {
			return
		} else if sent {
			ticker.Reset(s.activeSnapshotInterval)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !pollCompleted() {
					return
				}
				if keepGoing, sent := sendActive(); !keepGoing {
					return
				} else if sent {
					ticker.Reset(s.activeSnapshotInterval)
				}
			case event, ok := <-transitionSubscription.Events:
				if !ok {
					return
				}
				if eventOverlapsWindow(event, params.Window) {
					if !emitCompleted(eventFromTransition(event)) {
						return
					}
				}
				if keepGoing, sent := sendActive(); !keepGoing {
					return
				} else if sent {
					ticker.Reset(s.activeSnapshotInterval)
				}
			}
		}
	}()

	return Subscription{
		Events: events,
		Close: func() {
			cancel()
			transitionSubscription.Close()
		},
	}
}
