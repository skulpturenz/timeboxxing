package stack

import (
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment"
	appmetadata "github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/app_metadata"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/location"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// Stack returns the default enrichment pipeline and the OS permissions it needs.
// The caller (monitor.New) is responsible for requesting the permissions. Only
// the location enricher needs a permission today; app-metadata and browser do not.
func Stack() (enrichment.Enricher, []permission.Permission) {
	locationProvider := location.DefaultLocationProvider()

	enricher := enrichment.Pipe(
		enrichment.Or(appmetadata.Memoized(appmetadata.LocalMetadataEnricher),
			appmetadata.Memoized(appmetadata.FlathubEnricher),
			appmetadata.Memoized(appmetadata.WingetEnricher),
		),
		browser.Enrich(browser.NewCDPPoller(9222)),
		location.Enrich(locationProvider, location.NewPublicIPProvider()),
	)

	var permissions []permission.Permission
	if perm, ok := location.Requestable(locationProvider); ok {
		permissions = append(permissions, perm)
	}

	return enricher, permissions
}
