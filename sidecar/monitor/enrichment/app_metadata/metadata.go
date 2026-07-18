package appmetadata

import (
	"strings"
)

// KeyMetadata is the well-known key under which enrichers stash the typed
// Metadata payload inside ForegroundProcess.Enrichments.
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
