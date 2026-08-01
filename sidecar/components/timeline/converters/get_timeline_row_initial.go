package converters

import (
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
)

// A timeline row carries both endpoints of an entry side by side, so each side gets its own
// converter over the same row: goverter resolves sub-mappings by (source, target) signature, and
// two initial/final method sets on one converter would be an ambiguous match.

// goverter:converter
// goverter:name GetTimelineInitialRowConverter
// goverter:output:file ./get_timeline_row_initial.gen.go
// goverter:skipCopySameType
// goverter:useZeroValueOnPointerInconsistency
type getTimelineInitialRowConverter interface {
	// goverter:map InitialApplicationName AppName
	// goverter:map InitialApplicationIdentifier AppIdentifier
	// goverter:map InitialApplicationPath AppPath
	// goverter:map InitialPid PID
	// goverter:map InitialWindowTitle WindowTitle
	// goverter:map InitialTitleSource TitleSource | parseOptionalTitleSource
	// goverter:map InitialCreatedAtUtc Timestamp
	// goverter:map InitialIdle Idle
	// goverter:map InitialKilled Killed
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
	// goverter:map . Vendor | initialBrowserVendor
	// goverter:map InitialBrowserCategory Category | parseOptionalBrowserCategory
	// goverter:map InitialTab Tab
	// goverter:map InitialCdpUrl CdpURL
	toBrowser(source readqueries.GetTimelineRow) models.Browser

	// goverter:map InitialLatitude Latitude
	// goverter:map InitialLongitude Longitude
	// goverter:map InitialPublicIp PublicIP
	toLocation(source readqueries.GetTimelineRow) models.Location
}
