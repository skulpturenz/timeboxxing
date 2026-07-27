package converters

//go:generate go tool goverter gen .

import (
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	appmetadata "github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/app_metadata"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/location"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

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
// enums are distinct Go types but share identical iota ordering, so the numeric value is the mapping.
// (enumscategories.Parse cannot be used here — it lacks the "productivity" and "games" cases.)
func appmetadataCategory(category appmetadata.Category) enumscategories.Category {
	return enumscategories.Category(int(category))
}
