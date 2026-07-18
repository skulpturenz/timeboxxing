package memo

import (
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

type Memoize struct {
	cache *cache.Cache
	group singleflight.Group
}

const (
	defaultExpiration      = 250 * time.Millisecond
	defaultCleanupInterval = 500 * time.Millisecond
)

func New() *Memoize {
	c := cache.New(defaultExpiration, defaultCleanupInterval)

	return &Memoize{
		cache: c,
	}
}

func NewWithOptions(expiration *time.Duration, cleanupInterval *time.Duration) *Memoize {
	defaultExpiration := defaultExpiration
	if expiration != nil {
		defaultExpiration = *expiration
	}

	defaultCleanupInterval := defaultCleanupInterval
	if cleanupInterval != nil {
		defaultCleanupInterval = *cleanupInterval
	}

	c := cache.New(defaultExpiration, defaultCleanupInterval)

	return &Memoize{
		cache: c,
	}
}

func (m *Memoize) Do(key string, fn func() (any, error)) (any, error, bool) {
	v, ok := m.cache.Get(key)
	if ok {
		return v, nil, true
	}

	v, err, _ := m.group.Do(key, func() (any, error) {
		v, err := fn()
		if err == nil {
			m.cache.Set(key, v, cache.DefaultExpiration)
		}

		return v, err
	})

	return v, err, false
}

func (m *Memoize) Flush() {
	m.cache.Flush()
}
