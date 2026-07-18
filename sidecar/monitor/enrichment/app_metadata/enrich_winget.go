package appmetadata

import (
	"context"
	"net/url"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// wingetBaseURL is the winget.run search endpoint; overridable in tests.
var wingetBaseURL = "https://api.winget.run/v2/packages"

// wingetSearch is the subset of the winget.run search response we use.
type wingetSearch struct {
	Packages []struct {
		Latest struct {
			Name        string   `json:"Name"`
			Publisher   string   `json:"Publisher"`
			Description string   `json:"Description"`
			Tags        []string `json:"Tags"`
		} `json:"Latest"`
		IconURL string `json:"IconUrl"`
	} `json:"Packages"`
}

// WingetEnricher returns an enrichment.Enricher that fills gaps from the winget.run feed — mainly
// the description and a best-effort category (winget exposes only loose tags, no
// real taxonomy), which local Windows metadata cannot provide. Fill-if-empty;
// no-op when enabled is false. Compose after the local enrichment.Enricher and wrap in Memoized.
func WingetEnricher(ctx context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
	existing, _ := GetMetadata(fp)

	query := wingetQuery(existing, fp)
	if query == "" {
		return fp, false
	}

	endpoint := wingetBaseURL + "?query=" + url.QueryEscape(query) + "&limit=1&ensureContains=true"
	var payload wingetSearch
	ok, err := getJSON(ctx, endpoint, &payload)
	if err != nil || !ok || len(payload.Packages) == 0 {
		return fp, false
	}
	pkg := payload.Packages[0]

	metadata := Metadata{
		Source:       SourceWinget,
		FriendlyName: pkg.Latest.Name,
		Description:  pkg.Latest.Description,
	}
	if category, err := ParseWingetTags(pkg.Latest.Tags); err == nil {
		metadata.Category = category
	}
	if existing.IconPath == "" && pkg.IconURL != "" {
		metadata.IconPath = downloadIconToCache(ctx, pkg.IconURL, identityKey(fp))
	}

	if metadata.empty() {
		return fp, false
	}
	return setMetadata(fp, metadata)
}

// wingetQuery picks the best search term: the locally-resolved friendly name,
// falling back to the process's reported app name.
func wingetQuery(existing Metadata, fp monitor.ForegroundProcess) string {
	if name := strings.TrimSpace(existing.FriendlyName); name != "" {
		return name
	}
	if fp.AppName != nil {
		return strings.TrimSpace(*fp.AppName)
	}
	return ""
}
