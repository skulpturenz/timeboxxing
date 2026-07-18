package reporter

import (
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

func isReported(current *monitor.ForegroundProcess, incoming monitor.ForegroundProcess) bool {
	if current == nil {
		return false
	}

	if current.Idle != incoming.Idle {
		return false
	}

	if current.PID != nil && incoming.PID != nil {
		if *current.PID != *incoming.PID {
			return false
		}

		if deref(current.WindowTitle) != deref(incoming.WindowTitle) { // browsers
			return false
		}
	}

	return true
}

func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
