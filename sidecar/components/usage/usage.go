package usage

import (
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

const defaultActiveSnapshotInterval = 30 * time.Second

type Service struct {
	transitions            *componentTransitions.Service
	clock                  func() time.Time
	activeSnapshotInterval time.Duration
}

type serviceKey struct{}
type clockKey struct{}
type activeSnapshotIntervalKey struct{}

func RegisterService(registry *services.Services[any, any], service *Service) {
	services.Set(registry, serviceKey{}, service)
}

func ServiceFromServices(registry *services.Services[any, any]) (*Service, bool) {
	service, ok := services.Get[*Service](registry, serviceKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func RegisterClock(registry *services.Services[any, any], clock func() time.Time) {
	services.Set(registry, clockKey{}, clock)
}

func RegisterActiveSnapshotInterval(registry *services.Services[any, any], interval time.Duration) {
	services.Set(registry, activeSnapshotIntervalKey{}, interval)
}

func NewService(registry *services.Services[any, any]) *Service {
	transitions, _ := componentTransitions.ServiceFromServices(registry)
	clock := clockFromServices(registry)
	if clock == nil {
		clock = time.Now
	}
	interval := activeSnapshotIntervalFromServices(registry)
	if interval <= 0 {
		interval = defaultActiveSnapshotInterval
	}
	service := &Service{
		transitions:            transitions,
		clock:                  clock,
		activeSnapshotInterval: interval,
	}
	RegisterService(registry, service)
	return service
}

func minDuration(first time.Duration, second time.Duration) time.Duration {
	if first < second {
		return first
	}
	return second
}

func clockFromServices(registry *services.Services[any, any]) func() time.Time {
	service, ok := services.Get[func() time.Time](registry, clockKey{})
	if !ok {
		return nil
	}
	return service.Unwrap()
}

func activeSnapshotIntervalFromServices(registry *services.Services[any, any]) time.Duration {
	service, ok := services.Get[time.Duration](registry, activeSnapshotIntervalKey{})
	if !ok {
		return 0
	}
	return service.Unwrap()
}
