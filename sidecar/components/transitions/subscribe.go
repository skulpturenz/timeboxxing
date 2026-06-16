package transitions

import "context"

func (s *Service) Subscribe(ctx context.Context, params SubscribeParams) Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubscriberID++
	id := s.nextSubscriberID
	events := make(chan Event, subscriberBufferSize)
	s.subscribers[id] = subscriber{
		filters: params.Filters,
		events:  events,
	}

	subscription := Subscription{
		Events: events,
		Close: func() {
			s.unsubscribe(id)
		},
	}

	go func() {
		<-ctx.Done()
		subscription.Close()
	}()

	return subscription
}
