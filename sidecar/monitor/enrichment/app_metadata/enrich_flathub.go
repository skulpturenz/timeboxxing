package appmetadata

import (
	"context"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

var flathubBaseURL = "https://flathub.org/api/v2/appstream/"

type flathubAppstream struct {
	Name          string   `json:"name"`
	Summary       string   `json:"summary"`
	Description   string   `json:"description"`
	DeveloperName string   `json:"developer_name"`
	Categories    []string `json:"categories"`
	Icon          string   `json:"icon"`
}

func FlathubEnricher(ctx context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
	// An unconfigured base URL means the feed is disabled: no-op.
	if utils.IsEmptyString(flathubBaseURL) {
		return fp, false
	}

	appID := normalizeFlatpakAppId(fp)
	if appID == "" {
		return fp, false
	}

	var payload flathubAppstream
	ok, err := getJSON(ctx, flathubBaseURL+appID, &payload)
	if err != nil || !ok {
		return fp, false
	}

	metadata := Metadata{
		Source:       SourceFlathub,
		FriendlyName: payload.Name,
		Description: utils.
			Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
				payload.Summary,
				payload.Description), ""),
	}

	if category, err := ParseFreedesktopCategories(strings.Join(payload.Categories, ";")); err == nil {
		metadata.Category = category
	}

	if payload.Icon != "" {
		metadata.IconPath = downloadIconToCache(ctx, payload.Icon, identityKey(fp))
	}

	if metadata.empty() {
		return fp, false
	}

	updated := fp
	updated.Enrichments[KeyMetadata] = metadata
	return updated, true
}

func normalizeFlatpakAppId(fp monitor.ForegroundProcess) string {
	if fp.AppIdentifier == nil {
		return ""
	}

	id := strings.TrimSpace(*fp.AppIdentifier)
	if strings.Count(id, ".") < 2 {
		return ""
	}

	// A reverse-DNS id has no spaces or path separators.
	if strings.ContainsAny(id, " /\\") {
		return ""
	}

	return id
}
