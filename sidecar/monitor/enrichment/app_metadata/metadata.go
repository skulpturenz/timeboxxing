package appmetadata

import (
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// KeyMetadata is the well-known key under which enrichers stash the typed
// Metadata payload inside ForegroundProcess.Enrichments. Consumers read it
// with GetMetadata instead of poking the map[string]any directly.
const KeyMetadata = "appmetadata"

// Enrichment source tags, recorded on Metadata.Source so downstream code can
// tell where a field came from (local OS metadata vs a network feed).
const (
	SourceBundle  = "bundle"  // macOS .app Info.plist
	SourceDesktop = "desktop" // Linux .desktop entry
	SourcePE      = "pe"      // Windows PE version resource
	SourceFlathub = "flathub" // Flathub AppStream feed
	SourceWinget  = "winget"  // winget.run feed
)

// Metadata is the enriched, human-facing description of a foreground app.
// Empty fields mean "unknown" — an enricher only fills what it actually found.
type Metadata struct {
	FriendlyName string
	Description  string
	Category     Category // normalized taxonomy; CategoryUnknown when not resolved
	IconPath     string   // path to the cached icon file on disk
	Source       string   // origin of the metadata (see Source* constants)
}

// empty reports whether the metadata carries no information at all.
func (m Metadata) empty() bool {
	return strings.TrimSpace(m.FriendlyName) == "" &&
		strings.TrimSpace(m.Description) == "" &&
		m.Category == CategoryUnknown &&
		strings.TrimSpace(m.IconPath) == ""
}

// complete reports whether every field a feed enricher could fill is already
// populated. When true, a network feed has nothing to add and can be skipped.
func (m Metadata) complete() bool {
	return strings.TrimSpace(m.FriendlyName) != "" &&
		strings.TrimSpace(m.Description) != "" &&
		m.Category != CategoryUnknown &&
		strings.TrimSpace(m.IconPath) != ""
}

// mergeInto fills every field that is still empty in dst from src, and returns
// whether dst gained anything. Existing values win, so an earlier local enricher
// is never overwritten by a later feed fallback.
func (src Metadata) mergeInto(dst *Metadata) bool {
	changed := false
	fill := func(target *string, value string) {
		value = strings.TrimSpace(value)
		if *target == "" && value != "" {
			*target = value
			changed = true
		}
	}
	fill(&dst.FriendlyName, src.FriendlyName)
	fill(&dst.Description, src.Description)
	if dst.Category == CategoryUnknown && src.Category != CategoryUnknown {
		dst.Category = src.Category
		changed = true
	}
	fill(&dst.IconPath, src.IconPath)
	// Source names the first enricher that contributed anything.
	if changed && dst.Source == "" {
		dst.Source = strings.TrimSpace(src.Source)
	}
	return changed
}

// GetMetadata returns the Metadata currently stored on the process, plus
// whether it was present. It never panics on a nil or wrongly-typed bag entry.
func GetMetadata(fp monitor.ForegroundProcess) (Metadata, bool) {
	if fp.Enrichments == nil {
		return Metadata{}, false
	}
	value, ok := fp.Enrichments[KeyMetadata]
	if !ok {
		return Metadata{}, false
	}
	metadata, ok := value.(Metadata)
	return metadata, ok
}

// setMetadata merges found into whatever metadata the process already carries
// and writes the result back into the bag. It returns the (possibly copied)
// process and whether the merge added any new information. The input is treated
// as immutable: the Enrichments map is cloned before mutation.
func setMetadata(fp monitor.ForegroundProcess, found Metadata) (monitor.ForegroundProcess, bool) {
	existing, _ := GetMetadata(fp)
	if !found.mergeInto(&existing) {
		return fp, false
	}

	bag := make(map[string]any, len(fp.Enrichments)+1)
	for k, v := range fp.Enrichments {
		bag[k] = v
	}
	bag[KeyMetadata] = existing
	fp.Enrichments = bag
	return fp, true
}
