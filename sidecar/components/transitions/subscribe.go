package transitions

func (s *Service) Subscribe(params SubscribeParams) Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubscriberID++
	id := s.nextSubscriberID
	events := make(chan Event, subscriberBufferSize)
	s.subscribers[id] = subscriber{
		filters: params.Filters,
		events:  events,
	}

	return Subscription{
		Events: events,
		Close: func() {
			s.unsubscribe(id)
		},
	}
}
