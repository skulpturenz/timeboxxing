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
	// goverter:map TitleSource TitleSource | parseOptionalTitleSource
	// goverter:map CreatedAtUtc Timestamp
	// goverter:map . Enrichments
	ToForegroundProcess(source readqueries.GetUnindexedForegroundProcessesRow) models.ForegroundProcess

	// goverter:ignore Appmetadata
	// goverter:map . Browser
	// goverter:map . Location
	toEnrichments(source readqueries.GetUnindexedForegroundProcessesRow) models.Enrichments

	// goverter:ignore AppIdentifier Domain
	// goverter:map . Vendor | unindexedBrowserVendor
	// goverter:map BrowserCategory Category | parseOptionalBrowserCategory
	// goverter:map CdpUrl CdpURL
	toBrowser(source readqueries.GetUnindexedForegroundProcessesRow) models.Browser

	// goverter:map PublicIp PublicIP
	toLocation(source readqueries.GetUnindexedForegroundProcessesRow) models.Location
}
