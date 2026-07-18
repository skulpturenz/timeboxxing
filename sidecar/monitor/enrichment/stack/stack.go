package stack

import (
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment"
	appmetadata "github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/app_metadata"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/location"
)

func Stack() enrichment.Enricher {
	return enrichment.Pipe(
		enrichment.Or(appmetadata.Memoized(appmetadata.LocalMetadataEnricher),
			appmetadata.Memoized(appmetadata.FlathubEnricher),
			appmetadata.Memoized(appmetadata.WingetEnricher),
		),
		browser.Enrich(browser.NewCDPPoller(9222)),
		location.Default(),
	)
}
