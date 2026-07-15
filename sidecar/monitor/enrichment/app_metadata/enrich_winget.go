package appmetadata

import (
	"context"
	"net/url"
	"strings"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
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

// Winget returns an enricher that fills gaps from the winget.run feed — mainly
// the description and a best-effort category (winget exposes only loose tags, no
// real taxonomy), which local Windows metadata cannot provide. Fill-if-empty;
// no-op when enabled is false. Compose after the local enricher and wrap in Memoized.
func Winget(enabled bool) Enricher {
	if !enabled {
		return noopEnricher
	}
	return func(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		existing, _ := GetMetadata(fp)
		if !existing.needsFeedFill() {
			return fp, false
		}

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
		if code, label := categoryFromKeywords(pkg.Latest.Tags); code != "" {
			metadata.CategoryCode = code
			metadata.CategoryLabel = label
		}
		if existing.IconPath == "" && pkg.IconURL != "" {
			metadata.IconPath = downloadIconToCache(ctx, pkg.IconURL, identityKey(fp))
		}

		if metadata.empty() {
			return fp, false
		}
		return setMetadata(fp, metadata)
	}
}

// wingetQuery picks the best search term: the locally-resolved friendly name,
// falling back to the process's reported app name.
func wingetQuery(existing Metadata, fp sessionnew.ForegroundProcess) string {
	if name := strings.TrimSpace(existing.FriendlyName); name != "" {
		return name
	}
	if fp.AppName != nil {
		return strings.TrimSpace(*fp.AppName)
	}
	return ""
}

// keywordCategories maps common winget tag keywords to our normalized codes.
// winget tags are freeform, so this is a best-effort heuristic.
var keywordCategories = map[string]string{
	"developer":    CategoryDevelopment,
	"development":  CategoryDevelopment,
	"ide":          CategoryDevelopment,
	"editor":       CategoryDevelopment,
	"terminal":     CategoryDevelopment,
	"git":          CategoryDevelopment,
	"browser":      CategoryWebBrowsing,
	"chat":         CategoryCommunication,
	"messaging":    CategoryCommunication,
	"email":        CategoryCommunication,
	"video":        CategoryMedia,
	"audio":        CategoryMedia,
	"music":        CategoryMedia,
	"media":        CategoryMedia,
	"player":       CategoryMedia,
	"game":         CategoryGames,
	"gaming":       CategoryGames,
	"design":       CategoryGraphicsDesign,
	"graphics":     CategoryGraphicsDesign,
	"photo":        CategoryGraphicsDesign,
	"office":       CategoryProductivity,
	"productivity": CategoryProductivity,
	"note":         CategoryProductivity,
	"utility":      CategoryUtilities,
	"utilities":    CategoryUtilities,
	"social":       CategorySocial,
	"education":    CategoryEducation,
	"finance":      CategoryBusiness,
	"business":     CategoryBusiness,
}

// categoryFromKeywords scans tags for the first recognizable keyword.
func categoryFromKeywords(tags []string) (code string, label string) {
	for _, tag := range tags {
		key := strings.ToLower(strings.TrimSpace(tag))
		if key == "" {
			continue
		}
		if c, ok := keywordCategories[key]; ok {
			return c, categoryLabels[c]
		}
		// Also match multi-word tags by their individual words.
		for _, word := range strings.FieldsFunc(key, func(r rune) bool { return r == ' ' || r == '-' || r == '_' }) {
			if c, ok := keywordCategories[word]; ok {
				return c, categoryLabels[c]
			}
		}
	}
	return "", ""
}
