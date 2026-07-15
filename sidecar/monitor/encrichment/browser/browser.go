// Package browser provides an enricher that annotates a foreground process with
// browser tab information — the active browser, the tab/page title, and (when a
// Chrome DevTools endpoint is reachable) the full URL and its domain. It mirrors
// the detection logic in sidecar/monitor/browser.
package browser

import (
	"context"
	"net/url"
	"strings"

	browserkit "github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/encrichment"
	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

// Key is the Enrichments bag key under which the Tab payload is stored.
const Key = "browser"

// Tab describes the active browser tab for a foreground process. Fields are
// filled progressively: Browser is always set for a browser window; Title needs
// a parseable window title; URL/Domain need a reachable CDP endpoint.
type Tab struct {
	Browser string // "Chrome", "Firefox", "Safari", "Edge"
	Title   string // page/tab title, browser suffix stripped
	URL     string // full URL when resolvable (Chrome/Edge via CDP)
	Domain  string // host from URL without a leading "www.", e.g. "github.com"
}

// URLResolver resolves a tab title to its full URL. *browserkit.CDPPoller
// satisfies it; it is an interface so callers can pass nil or a fake in tests.
type URLResolver interface {
	URLForTitle(ctx context.Context, tabTitle string) string
}

// Enrich returns an enricher that tags browser windows with their tab info. The
// resolver (typically a *browser.CDPPoller) supplies the URL for Chrome/Edge; it
// may be nil, in which case URL and Domain are left empty. Unlike the app
// metadata enrichers this is NOT memoized — the tab changes constantly, so it
// runs every poll (the CDP poller rate-limits its own network calls).
func Enrich(resolver URLResolver) encrichment.Enricher {
	return func(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		appName := deref(fp.AppName)
		kind := browserkit.IsBrowser(appName)
		if kind == browserkit.BrowserNone {
			return fp, false
		}

		tab := Tab{Browser: kind.String()}

		if title := deref(fp.WindowTitle); title != "" {
			if info, ok := browserkit.ParseTabTitle(appName, title); ok {
				tab.Title = info.TabTitle
				if resolver != nil {
					if resolved := resolver.URLForTitle(ctx, info.TabTitle); resolved != "" {
						tab.URL = resolved
						tab.Domain = domainOf(resolved)
					}
				}
			}
		}

		return setTab(fp, tab), true
	}
}

// Get returns the browser Tab stored on the process, if any.
func Get(fp sessionnew.ForegroundProcess) (Tab, bool) {
	if fp.Enrichments == nil {
		return Tab{}, false
	}
	value, ok := fp.Enrichments[Key]
	if !ok {
		return Tab{}, false
	}
	tab, ok := value.(Tab)
	return tab, ok
}

// setTab writes tab into a cloned Enrichments bag (the input is treated as
// immutable) and returns the updated process.
func setTab(fp sessionnew.ForegroundProcess, tab Tab) sessionnew.ForegroundProcess {
	bag := make(map[string]any, len(fp.Enrichments)+1)
	for k, v := range fp.Enrichments {
		bag[k] = v
	}
	bag[Key] = tab
	fp.Enrichments = bag
	return fp
}

// domainOf extracts the host (minus a leading "www.") from an http(s) URL.
// Non-web schemes (chrome://, about:, file://) yield "".
func domainOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
