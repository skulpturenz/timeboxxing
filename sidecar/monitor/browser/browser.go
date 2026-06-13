package browser

import "strings"

// BrowserKind identifies a supported browser.
type BrowserKind int

const (
	BrowserNone    BrowserKind = iota
	BrowserChrome              // Google Chrome, Chromium, Chrome Canary
	BrowserFirefox             // Mozilla Firefox, Firefox Developer Edition
	BrowserSafari              // Apple Safari
	BrowserEdge                // Microsoft Edge
)

func (k BrowserKind) String() string {
	switch k {
	case BrowserChrome:
		return "Chrome"
	case BrowserFirefox:
		return "Firefox"
	case BrowserSafari:
		return "Safari"
	case BrowserEdge:
		return "Edge"
	default:
		return ""
	}
}

// browserAppNames maps lowercase app names to BrowserKind.
// Covers macOS localizedName, Windows exe-derived names, Linux /proc/pid/comm.
var browserAppNames = map[string]BrowserKind{
	// macOS / Windows display names
	"google chrome":              BrowserChrome,
	"google chrome canary":       BrowserChrome,
	"chromium":                   BrowserChrome,
	"chromium-browser":           BrowserChrome,
	"firefox":                    BrowserFirefox,
	"firefox developer edition":  BrowserFirefox,
	"firefox nightly":            BrowserFirefox,
	"safari":                     BrowserSafari,
	"safari technology preview":  BrowserSafari,
	"microsoft edge":             BrowserEdge,
	"microsoft edge canary":      BrowserEdge,
	// Windows exe-derived (after stripping .exe and lowercasing)
	"chrome":        BrowserChrome,
	"msedge":        BrowserEdge,
	"microsoft-edge": BrowserEdge,
	// Linux /proc/<pid>/comm (max 15 chars, may be truncated)
	"google-chrome":  BrowserChrome,
	"google-chrome-": BrowserChrome, // truncated
	"chromium-browse": BrowserChrome,
}

// IsBrowser returns the BrowserKind for the given app name, or BrowserNone.
func IsBrowser(appName string) BrowserKind {
	if k, ok := browserAppNames[strings.ToLower(strings.TrimSpace(appName))]; ok {
		return k
	}
	// Prefix match for truncated Linux process names.
	lower := strings.ToLower(appName)
	if strings.HasPrefix(lower, "google-chrome") || strings.HasPrefix(lower, "chromium") {
		return BrowserChrome
	}
	if strings.HasPrefix(lower, "firefox") {
		return BrowserFirefox
	}
	if strings.HasPrefix(lower, "msedge") || strings.HasPrefix(lower, "microsoft-edge") {
		return BrowserEdge
	}
	return BrowserNone
}

// TabInfo is the result of parsing a raw browser window title.
type TabInfo struct {
	Browser   BrowserKind
	TabTitle  string // page title, browser suffix stripped
	RawTitle  string // original unmodified window title
}

// browserSuffixes lists the suffixes that each browser appends to window titles.
// Tried longest-first so the most specific match wins.
var browserSuffixes = map[BrowserKind][]string{
	BrowserChrome: {
		" \u2014 Google Chrome",  // em-dash variant (some locales)
		" - Google Chrome",
		" \u2014 Chromium",
		" - Chromium",
		" \u2014 Google Chrome Canary",
		" - Google Chrome Canary",
	},
	BrowserFirefox: {
		" \u2014 Mozilla Firefox",
		" - Mozilla Firefox",
		" \u2014 Firefox",
		" - Firefox",
		" \u2014 Firefox Developer Edition",
		" - Firefox Developer Edition",
		" \u2014 Firefox Nightly",
		" - Firefox Nightly",
	},
	BrowserSafari: {
		// Safari does not append its name to the window title.
		// The entire window title is the tab/page title.
	},
	BrowserEdge: {
		" \u2014 Microsoft Edge",
		" - Microsoft Edge",
		" \u2014 Microsoft Edge Canary",
		" - Microsoft Edge Canary",
	},
}

// ParseTabTitle strips the browser suffix from rawTitle and returns a TabInfo.
// Returns (TabInfo{}, false) if the title is empty or cannot be parsed as a tab
// (e.g., a native browser dialog with no recognisable suffix).
//
// For Safari, any non-empty title is accepted as a tab title because Safari does
// not include its own name in window titles.
func ParseTabTitle(appName, rawTitle string) (TabInfo, bool) {
	if rawTitle == "" {
		return TabInfo{}, false
	}
	kind := IsBrowser(appName)
	if kind == BrowserNone {
		return TabInfo{}, false
	}

	// Safari: full title IS the tab title.
	if kind == BrowserSafari {
		return TabInfo{Browser: kind, TabTitle: rawTitle, RawTitle: rawTitle}, true
	}

	suffixes := browserSuffixes[kind]
	for _, suffix := range suffixes {
		if strings.HasSuffix(rawTitle, suffix) {
			tab := strings.TrimSuffix(rawTitle, suffix)
			if tab == "" {
				// Title was exactly the browser name — probably a blank new-tab page.
				tab = "New Tab"
			}
			return TabInfo{Browser: kind, TabTitle: tab, RawTitle: rawTitle}, true
		}
	}

	// Special case: rawTitle is exactly the browser name with no page prefix.
	// This happens when Chrome/Edge/Firefox is loading or showing the new-tab page
	// without a suffix.
	lower := strings.ToLower(rawTitle)
	knownNames := map[BrowserKind][]string{
		BrowserChrome:  {"google chrome", "chromium"},
		BrowserFirefox: {"mozilla firefox", "firefox"},
		BrowserEdge:    {"microsoft edge"},
	}
	for _, name := range knownNames[kind] {
		if lower == name {
			return TabInfo{Browser: kind, TabTitle: "New Tab", RawTitle: rawTitle}, true
		}
	}

	// No suffix matched — likely a native browser dialog (download manager, devtools
	// opened in the same window, etc.). Treat as not a trackable tab.
	return TabInfo{}, false
}
