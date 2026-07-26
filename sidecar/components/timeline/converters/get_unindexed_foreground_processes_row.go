package converters

import (
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
)

// goverter:converter
// goverter:name GetUnindexedForegroundProcessesRowConverter
// goverter:output:file ./get_unindexed_foreground_processes_row.gen.go
// goverter:skipCopySameType
// goverter:useZeroValueOnPointerInconsistency
type getUnindexedForegroundProcessesRowConverter interface {
	// goverter:ignore AppPath
	// goverter:map ApplicationName AppName | zeroNilString
	// goverter:map ApplicationIdentifier AppIdentifier
	// goverter:map Pid PID
	// goverter:map TitleSource TitleSource | parseTitleSource
	// goverter:map CreatedAtUtc Timestamp
	// goverter:map . Enrichments
	ToForegroundProcess(source readqueries.GetUnindexedForegroundProcessesRow) models.ForegroundProcess

	// goverter:ignore Appmetadata
	// goverter:map . Browser
	// goverter:map . Location
	toEnrichments(source readqueries.GetUnindexedForegroundProcessesRow) models.Enrichments

	// goverter:ignore AppIdentifier CdpURL Domain
	// goverter:map BrowserVendor Vendor
	// goverter:map BrowserCategory Category | parseBrowserCategory
	toBrowser(source readqueries.GetUnindexedForegroundProcessesRow) models.Browser

	// goverter:map PublicIp PublicIP
	toLocation(source readqueries.GetUnindexedForegroundProcessesRow) models.Location
}
