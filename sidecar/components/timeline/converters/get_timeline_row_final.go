package converters

import (
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
)

// The final side of a timeline row is absent while the entry is still open — callers must gate on
// GetTimelineRow.FinalFpID before converting.

// goverter:converter
// goverter:name GetTimelineFinalRowConverter
// goverter:output:file ./get_timeline_row_final.gen.go
// goverter:skipCopySameType
// goverter:useZeroValueOnPointerInconsistency
type getTimelineFinalRowConverter interface {
	// goverter:map FinalApplicationName AppName
	// goverter:map FinalApplicationIdentifier AppIdentifier
	// goverter:map FinalApplicationPath AppPath
	// goverter:map FinalPid PID
	// goverter:map FinalWindowTitle WindowTitle
	// goverter:map FinalTitleSource TitleSource | parseOptionalTitleSource
	// goverter:map FinalCreatedAtUtc Timestamp
	// goverter:map FinalIdle Idle
	// goverter:map FinalKilled Killed
	// goverter:map . Enrichments
	ToForegroundProcess(source readqueries.GetTimelineRow) models.ForegroundProcess

	// goverter:map . Appmetadata
	// goverter:map . Browser
	// goverter:map . Location
	toEnrichments(source readqueries.GetTimelineRow) models.Enrichments

	// Nothing about the application is on the row: the enrichment is resolved live by the monitor
	// and never written to the event store, and the one persisted part — the category — is looked
	// up per application and merged in afterwards by ApplicationCategoriesConverter.
	// goverter:ignore FriendlyName Description IconPath Source Category
	toAppMetadata(source readqueries.GetTimelineRow) models.AppMetadata

	// goverter:ignore AppIdentifier Domain
	// goverter:map . Vendor | finalBrowserVendor
	// goverter:map FinalBrowserCategory Category | parseOptionalBrowserCategory
	// goverter:map FinalTab Tab
	// goverter:map FinalCdpUrl CdpURL
	toBrowser(source readqueries.GetTimelineRow) models.Browser

	// goverter:map FinalLatitude Latitude
	// goverter:map FinalLongitude Longitude
	// goverter:map FinalPublicIp PublicIP
	toLocation(source readqueries.GetTimelineRow) models.Location
}
