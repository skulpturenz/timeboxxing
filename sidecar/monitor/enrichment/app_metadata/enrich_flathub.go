package appmetadata

import (
	"context"
	"strings"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

// flathubBaseURL is the Flathub AppStream endpoint; overridable in tests.
var flathubBaseURL = "https://flathub.org/api/v2/appstream/"

// flathubAppstream is the subset of the Flathub AppStream response we use.
type flathubAppstream struct {
	Name          string   `json:"name"`
	Summary       string   `json:"summary"`
	Description   string   `json:"description"`
	DeveloperName string   `json:"developer_name"`
	Categories    []string `json:"categories"`
	Icon          string   `json:"icon"`
}

// Flathub returns an enricher that fills gaps from the Flathub AppStream feed for
// Flatpak apps, whose Wayland app_id matches the Flathub app id (reverse-DNS,
// e.g. org.videolan.VLC). It only runs when local extraction left a field empty
// (fill-if-empty), so it is a fallback. When enabled is false it is a no-op.
// Compose it after the local enricher and wrap with Memoized:
//
//	enrichment.Pipe(Memoized(LocalMetadata), Memoized(Flathub(cfg.Feeds)))
func Flathub(enabled bool) Enricher {
	if !enabled {
		return noopEnricher
	}
	return func(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		appID := flatpakAppID(fp)
		if appID == "" {
			return fp, false
		}
		existing, _ := GetMetadata(fp)
		if !existing.needsFeedFill() {
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
			Description:  firstNonBlankFeed(payload.Summary, payload.Description),
		}
		if code, label := CategoryFromFreedesktop(strings.Join(payload.Categories, ";")); code != "" {
			metadata.CategoryCode = code
			metadata.CategoryLabel = label
		}
		if existing.IconPath == "" && payload.Icon != "" {
			metadata.IconPath = downloadIconToCache(ctx, payload.Icon, identityKey(fp))
		}

		if metadata.empty() {
			return fp, false
		}
		return setMetadata(fp, metadata)
	}
}

// flatpakAppID returns the app's identifier when it looks like a Flathub app id
// (reverse-DNS with at least two dots), else "".
func flatpakAppID(fp sessionnew.ForegroundProcess) string {
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

// needsFeedFill reports whether a feed could still contribute something.
func (m Metadata) needsFeedFill() bool {
	return strings.TrimSpace(m.FriendlyName) == "" ||
		strings.TrimSpace(m.Description) == "" ||
		strings.TrimSpace(m.CategoryCode) == "" ||
		strings.TrimSpace(m.IconPath) == ""
}

func firstNonBlankFeed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func noopEnricher(_ context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
	return fp, false
}
