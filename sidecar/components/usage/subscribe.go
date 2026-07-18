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

		sendActive := func() (bool, bool) {
			event, ok := s.activeEvent(ctx, params.Window)
			if !ok {
				return true, false
			}
			select {
			case events <- event:
				return true, true
			case <-ctx.Done():
				return false, false
			}
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
				if keepGoing, sent := sendActive(); !keepGoing {
					return
				} else if sent {
					ticker.Reset(s.activeSnapshotInterval)
				}
			case event, ok := <-transitionSubscription.Events:
				if !ok {
					return
				}
				if !eventOverlapsWindow(event, params.Window) {
					if keepGoing, sent := sendActive(); !keepGoing {
						return
					} else if sent {
						ticker.Reset(s.activeSnapshotInterval)
					}
					continue
				}
				select {
				case events <- eventFromTransition(event):
				case <-ctx.Done():
					return
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
