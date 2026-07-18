package reporter

import (
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
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

		if utils.Coalesce(current.WindowTitle, "") != utils.Coalesce(incoming.WindowTitle, "") { // browsers
			return false
		}
	}

	return true
}
