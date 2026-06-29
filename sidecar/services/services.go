package services

import (
	"iter"
	"sync"
)

type Service[T any] struct {
	value T
}

func (s Service[T]) Unwrap() T {
	return s.value
}

type Services[K comparable, V any] struct {
	mu       sync.RWMutex
	services map[K]V
}

func New() *Services[any, any] {
	return &Services[any, any]{
		services: make(map[any]any),
	}
}

func Set[K comparable, V any](svc *Services[any, any], key K, value V) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.services[key] = value
}

func Get[V any, K comparable](svc *Services[any, any], key K) (Service[V], bool) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	val, ok := svc.services[key]
	if !ok {
		return Service[V]{}, false
	}

	return Service[V]{value: val.(V)}, true
}

func (s *Services[K, V]) Entries() iter.Seq2[any, any] {
	s.mu.RLock()

	return func(yield func(any, any) bool) {
		defer s.mu.RUnlock()
		for idx, item := range s.services {
			if !yield(idx, item) {
				return
			}
		}
	}
}
