package converters

import (
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// goverter:converter
// goverter:name MonitorForegroundProcessConverter
// goverter:output:file ./monitor_foreground_process.gen.go
// goverter:skipCopySameType
// goverter:useZeroValueOnPointerInconsistency
type monitorForegroundProcessConverter interface {
	// goverter:ignore TitleSource Killed
	// goverter:map PID PID | pidInt32ToInt64
	// goverter:map . Enrichments | monitorEnrichments
	ToForegroundProcess(source monitor.ForegroundProcess) models.ForegroundProcess
}
