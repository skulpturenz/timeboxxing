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

		currentWindowTitle := *current.WindowTitle
		incomingWindowTitle := *incoming.WindowTitle

		if currentWindowTitle != incomingWindowTitle { // browsers
			return false
		}
	}

	return true
}
