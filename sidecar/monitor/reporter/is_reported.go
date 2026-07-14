package reporter

import sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"

func isReported(current *sessionnew.ForegroundProcess, incoming sessionnew.ForegroundProcess) bool {
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

	return true // dedupe by default
}
