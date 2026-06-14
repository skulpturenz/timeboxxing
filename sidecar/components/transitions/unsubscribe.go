package transitions

func (s *Service) unsubscribe(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, id)
}
