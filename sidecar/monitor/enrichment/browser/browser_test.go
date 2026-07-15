package browser

import (
	"context"
	"testing"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

type fakeResolver struct {
	url    string
	titles []string
}

func (f *fakeResolver) URLForTitle(_ context.Context, tabTitle string) string {
	f.titles = append(f.titles, tabTitle)
	return f.url
}

func ptr(s string) *string { return &s }

func fp(appName, title string) sessionnew.ForegroundProcess {
	return sessionnew.ForegroundProcess{
		AppName:     ptr(appName),
		WindowTitle: ptr(title),
		Enrichments: map[string]any{},
	}
}

func TestEnrich_ChromeWithURL(t *testing.T) {
	resolver := &fakeResolver{url: "https://github.com/foo/bar?x=1"}

	out, ok := Enrich(resolver)(context.Background(), fp("Google Chrome", "GitHub · foo - Google Chrome"))
	if !ok {
		t.Fatal("expected enrichment for a Chrome window")
	}
	tab, ok := Get(out)
	if !ok {
		t.Fatal("tab not stored in bag")
	}
	if tab.Browser != "Chrome" {
		t.Errorf("Browser = %q, want Chrome", tab.Browser)
	}
	if tab.Title != "GitHub · foo" {
		t.Errorf("Title = %q, want 'GitHub · foo'", tab.Title)
	}
	if tab.URL != "https://github.com/foo/bar?x=1" {
		t.Errorf("URL = %q", tab.URL)
	}
	if tab.Domain != "github.com" {
		t.Errorf("Domain = %q, want github.com", tab.Domain)
	}
	if len(resolver.titles) != 1 || resolver.titles[0] != "GitHub · foo" {
		t.Errorf("resolver queried with %v, want [GitHub · foo]", resolver.titles)
	}
}

func TestEnrich_NonBrowser(t *testing.T) {
	_, ok := Enrich(nil)(context.Background(), fp("Visual Studio Code", "main.go — myproj"))
	if ok {
		t.Error("expected no enrichment for a non-browser app")
	}
}

func TestEnrich_BrowserWithoutResolver(t *testing.T) {
	// A browser window with a parseable title but no CDP resolver: Browser and
	// Title populate, URL/Domain stay empty.
	out, ok := Enrich(nil)(context.Background(), fp("Firefox", "Wikipedia — Mozilla Firefox"))
	if !ok {
		t.Fatal("expected enrichment")
	}
	tab, _ := Get(out)
	if tab.Browser != "Firefox" || tab.Title != "Wikipedia" {
		t.Errorf("got %+v", tab)
	}
	if tab.URL != "" || tab.Domain != "" {
		t.Errorf("expected no URL/Domain without a resolver, got %+v", tab)
	}
}

func TestEnrich_BrowserUnparseableTitle(t *testing.T) {
	// A recognised browser but the title has no tab suffix (e.g. a devtools or
	// download window): still enriched with Browser, but no Title/URL.
	out, ok := Enrich(nil)(context.Background(), fp("Google Chrome", "DevTools"))
	if !ok {
		t.Fatal("expected enrichment for a browser window")
	}
	tab, _ := Get(out)
	if tab.Browser != "Chrome" {
		t.Errorf("Browser = %q", tab.Browser)
	}
	if tab.Title != "" {
		t.Errorf("Title = %q, want empty (unparseable)", tab.Title)
	}
}

func TestDomainOf(t *testing.T) {
	cases := map[string]string{
		"https://github.com/foo":       "github.com",
		"https://www.google.com/search": "google.com",
		"http://localhost:3000/x":      "localhost",
		"https://sub.example.co.uk":    "sub.example.co.uk",
		"chrome://newtab/":             "",
		"about:blank":                  "",
		"":                             "",
		"not a url":                    "",
	}
	for in, want := range cases {
		if got := domainOf(in); got != want {
			t.Errorf("domainOf(%q) = %q, want %q", in, got, want)
		}
	}
}
