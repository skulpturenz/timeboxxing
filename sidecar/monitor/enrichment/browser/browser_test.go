package browser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

type fakeResolver struct {
	url    string
	titles []string
}

func (f *fakeResolver) URLForTitle(_ context.Context, tabTitle string) string {
	f.titles = append(f.titles, tabTitle)
	return f.url
}

func fp(appName, title string) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{
		AppName:     new(appName),
		WindowTitle: new(title),
		Enrichments: map[string]any{},
	}
}

func TestEnrich_ChromeWithURL(t *testing.T) {
	resolver := &fakeResolver{url: "https://github.com/foo/bar?x=1"}

	out, ok := Enrich(resolver)(context.Background(), fp("Google Chrome", "GitHub · foo - Google Chrome"))
	require.True(t, ok, "expected enrichment for a Chrome window")
	tab, ok := Get(out)
	require.True(t, ok, "tab not stored in bag")

	assert.Equal(t, "Chrome", tab.Browser)
	assert.Equal(t, "GitHub · foo", tab.Title)
	assert.Equal(t, "https://github.com/foo/bar?x=1", tab.URL)
	assert.Equal(t, "github.com", tab.Domain)
	assert.Equal(t, []string{"GitHub · foo"}, resolver.titles, "resolver queried with unexpected titles")
}

func TestEnrich_NonBrowser(t *testing.T) {
	_, ok := Enrich(nil)(context.Background(), fp("Visual Studio Code", "main.go — myproj"))
	assert.False(t, ok, "expected no enrichment for a non-browser app")
}

func TestEnrich_BrowserWithoutResolver(t *testing.T) {
	// A browser window with a parseable title but no CDP resolver: Browser and
	// Title populate, URL/Domain stay empty.
	out, ok := Enrich(nil)(context.Background(), fp("Firefox", "Wikipedia — Mozilla Firefox"))
	require.True(t, ok, "expected enrichment")
	tab, _ := Get(out)
	assert.Equal(t, "Firefox", tab.Browser)
	assert.Equal(t, "Wikipedia", tab.Title)
	assert.Empty(t, tab.URL, "expected no URL without a resolver")
	assert.Empty(t, tab.Domain, "expected no Domain without a resolver")
}

func TestEnrich_BrowserUnparseableTitle(t *testing.T) {
	// A recognised browser but the title has no tab suffix (e.g. a devtools or
	// download window): still enriched with Browser, but no Title/URL.
	out, ok := Enrich(nil)(context.Background(), fp("Google Chrome", "DevTools"))
	require.True(t, ok, "expected enrichment for a browser window")
	tab, _ := Get(out)
	assert.Equal(t, "Chrome", tab.Browser)
	assert.Empty(t, tab.Title, "Title should be empty (unparseable)")
}

func TestDomainOf(t *testing.T) {
	cases := map[string]string{
		"https://github.com/foo":        "github.com",
		"https://www.google.com/search": "google.com",
		"http://localhost:3000/x":       "localhost",
		"https://sub.example.co.uk":     "sub.example.co.uk",
		"chrome://newtab/":              "",
		"about:blank":                   "",
		"":                              "",
		"not a url":                     "",
	}
	for in, want := range cases {
		assert.Equalf(t, want, domainOf(in), "domainOf(%q)", in)
	}
}
