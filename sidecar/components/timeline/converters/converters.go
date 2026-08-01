package converters

//go:generate go tool goverter gen .

import (
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	appmetadata "github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/app_metadata"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/location"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

// unknownBrowserVendor stands in for the vendor of an observation the event store flagged as a
// browser but never enriched, so the enrichment stays non-zero. See initialBrowserVendor.
const unknownBrowserVendor = "unknown"

func zeroNilString(v string) *string {
	return utils.ZeroNil(v)
}

func parseTitleSource(v *string) *models.TitleSource {
	titleSource, err := models.ParseTitleSource(utils.Coalesce(v, ""))
	assert.NoError(err)

	return utils.ZeroNil(titleSource)
}

func parseBrowserCategory(v string) *enumscategories.Category {
	category, err := enumscategories.Parse(v)
	assert.NoError(err)

	return utils.ZeroNil(category)
}

// parseOptionalTitleSource is parseTitleSource for sources where the column is genuinely absent
// rather than unrecognized — an idle observation has no window and so no title source.
func parseOptionalTitleSource(v *string) *models.TitleSource {
	if utils.Coalesce(v, "") == "" {
		return nil
	}

	return parseTitleSource(v)
}

// parseOptionalBrowserCategory is parseBrowserCategory for left-joined category codes, where an
// absent code means "no category" — a non-browser observation has no browser category.
func parseOptionalBrowserCategory(v *string) *enumscategories.Category {
	if utils.Coalesce(v, "") == "" {
		return nil
	}

	return parseBrowserCategory(*v)
}

// firstCategory collapses an application's classifications to the single category the model
// carries. The link is many-to-many, but ingest writes exactly one today; an application that has
// not been classified yet stays CategoryUnknown.
func firstCategory(categories []enumscategories.Category) enumscategories.Category {
	if len(categories) == 0 {
		return enumscategories.CategoryUnknown
	}

	return categories[0]
}

// initialBrowserVendor resolves the browser vendor of a timeline row's initial observation, falling
// back to a sentinel when the row is flagged as a browser but has not been enriched with a vendor
// yet. ForegroundProcess.IsBrowser() reports on the zero-ness of the whole Browser enrichment, so
// without the fallback an unenriched browser observation would read back as a plain application.
func initialBrowserVendor(row readqueries.GetTimelineRow) string {
	return browserVendor(row.InitialBrowser, row.InitialBrowserVendor)
}

// finalBrowserVendor is initialBrowserVendor for the final side of a timeline row.
func finalBrowserVendor(row readqueries.GetTimelineRow) string {
	return browserVendor(row.FinalBrowser, row.FinalBrowserVendor)
}

func browserVendor(browser *bool, vendor *string) string {
	if resolved := utils.Coalesce(vendor, ""); resolved != "" {
		return resolved
	}

	if utils.Coalesce(browser, false) {
		return unknownBrowserVendor
	}

	return ""
}

// pidInt32ToInt64 widens the monitor PID (*int32) to the model PID (*int64).
func pidInt32ToInt64(pid *int32) *int64 {
	if pid == nil {
		return nil
	}
	widened := int64(*pid)
	return &widened
}

// monitorEnrichments reads the app-metadata, browser, and location enrichments off the monitor
// process's dynamic enrichment bag (via the accessors that populate them) into the typed model.
func monitorEnrichments(fp monitor.ForegroundProcess) models.Enrichments {
	enrichments := models.Enrichments{}
	if md, ok := appmetadata.Get(fp); ok {
		enrichments.Appmetadata = models.AppMetadata{
			FriendlyName: md.FriendlyName,
			Description:  md.Description,
			Category:     appmetadataCategory(md.Category),
			IconPath:     md.IconPath,
			Source:       md.Source,
		}
	}
	if tab, ok := browser.Get(fp); ok {
		enrichments.Browser = models.Browser{
			Vendor: tab.Browser,
			Tab:    tab.Title,
			CdpURL: tab.URL,
			Domain: tab.Domain,
		}
	}
	if env, ok := location.Get(fp); ok {
		enrichments.Location = models.Location{
			Latitude:  env.Latitude,
			Longitude: env.Longitude,
			PublicIP:  env.PublicIP,
		}
	}
	return enrichments
}

// appmetadataCategory maps the app-metadata taxonomy to the shared enums_categories taxonomy. The two
// enums are distinct Go types but share identical iota ordering, so the numeric value is the mapping —
// the source is a sibling enum rather than a code, so there is nothing to parse.
func appmetadataCategory(category appmetadata.Category) enumscategories.Category {
	return enumscategories.Category(int(category))
}
