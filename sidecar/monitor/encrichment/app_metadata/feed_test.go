package app_metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/encrichment"
	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

// TestFlathub_Live hits the real Flathub endpoint to confirm the response still
// decodes into our struct. Opt-in (network): set TBX_LIVE_FEEDS=1 to run.
func TestFlathub_Live(t *testing.T) {
	if os.Getenv("TBX_LIVE_FEEDS") == "" {
		t.Skip("set TBX_LIVE_FEEDS=1 to run live feed tests")
	}
	out, ok := Flathub(true)(context.Background(), fpWithID("org.videolan.VLC"))
	if !ok {
		t.Fatal("expected live Flathub enrichment for VLC")
	}
	md, _ := GetMetadata(out)
	if md.FriendlyName == "" || md.CategoryCode == "" {
		t.Fatalf("live response under-populated: %+v", md)
	}
	t.Logf("live VLC => name=%q category=%q desc.len=%d", md.FriendlyName, md.CategoryLabel, len(md.Description))
}

func ptr(s string) *string { return &s }

func fpWithID(id string) sessionnew.ForegroundProcess {
	return sessionnew.ForegroundProcess{
		AppIdentifier: ptr(id),
		Enrichments:   map[string]any{},
	}
}

const vlcAppstreamJSON = `{
  "name": "VLC",
  "summary": "VLC media player, the open-source multimedia player",
  "description": "<p>VLC is a free and open source cross-platform multimedia player.</p>",
  "developer_name": "VideoLAN et al.",
  "categories": ["AudioVideo", "Player", "Recorder"],
  "icon": ""
}`

func TestFlathub_FillsFromFeed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/org.videolan.VLC" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(vlcAppstreamJSON))
	}))
	defer srv.Close()

	old := flathubBaseURL
	flathubBaseURL = srv.URL + "/"
	defer func() { flathubBaseURL = old }()

	out, ok := Flathub(true)(context.Background(), fpWithID("org.videolan.VLC"))
	if !ok {
		t.Fatal("expected Flathub to enrich")
	}
	md, _ := GetMetadata(out)
	if md.FriendlyName != "VLC" {
		t.Errorf("FriendlyName = %q", md.FriendlyName)
	}
	if md.CategoryCode != CategoryMedia {
		t.Errorf("CategoryCode = %q, want %q", md.CategoryCode, CategoryMedia)
	}
	if md.Source != SourceFlathub {
		t.Errorf("Source = %q", md.Source)
	}
	if md.Description == "" {
		t.Error("expected a description from the summary")
	}
}

func TestFlathub_SkipsWhenLocalComplete(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(vlcAppstreamJSON))
	}))
	defer srv.Close()
	old := flathubBaseURL
	flathubBaseURL = srv.URL + "/"
	defer func() { flathubBaseURL = old }()

	// Local enrichment already filled everything -> feed must not fire.
	fp := fpWithID("org.videolan.VLC")
	fp, _ = setMetadata(fp, Metadata{
		FriendlyName: "VLC", Description: "d", CategoryCode: CategoryMedia,
		CategoryLabel: "Media & Entertainment", IconPath: "/tmp/x.png", Source: SourceDesktop,
	})

	_, ok := Flathub(true)(context.Background(), fp)
	if ok {
		t.Error("expected no-op when metadata already complete")
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Errorf("feed was hit %d times, want 0", got)
	}
}

func TestFlathub_Disabled(t *testing.T) {
	_, ok := Flathub(false)(context.Background(), fpWithID("org.videolan.VLC"))
	if ok {
		t.Error("disabled Flathub should be a no-op")
	}
}

func TestFlathub_IgnoresNonReverseDNS(t *testing.T) {
	// "chrome" (Windows-style exe id) is not a flatpak app id.
	_, ok := Flathub(true)(context.Background(), fpWithID("chrome"))
	if ok {
		t.Error("expected no-op for non reverse-DNS identifier")
	}
}

func TestMemoized_MemoizesByIdentity(t *testing.T) {
	var calls int32
	inner := func(_ context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		atomic.AddInt32(&calls, 1)
		return setMetadata(fp, Metadata{FriendlyName: "X", Source: SourceBundle})
	}
	enricher := Memoized(inner)

	for i := 0; i < 3; i++ {
		out, ok := enricher(context.Background(), fpWithID("com.example.app"))
		if !ok {
			t.Fatal("expected enrichment")
		}
		if md, _ := GetMetadata(out); md.FriendlyName != "X" {
			t.Fatalf("FriendlyName = %q", md.FriendlyName)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("inner called %d times, want 1 (memoized)", got)
	}
}

func TestFold_MergesLocalThenFeed(t *testing.T) {
	local := func(_ context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		return setMetadata(fp, Metadata{FriendlyName: "VLC", IconPath: "/i.png", Source: SourceDesktop})
	}
	feed := func(_ context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
		// Feed only fills category/description; must not overwrite the name.
		return setMetadata(fp, Metadata{FriendlyName: "WRONG", Description: "desc", CategoryCode: CategoryMedia, Source: SourceFlathub})
	}

	out, ok := encrichment.Fold(local, feed)(context.Background(), fpWithID("org.videolan.VLC"))
	if !ok {
		t.Fatal("expected combined enrichment")
	}
	md, _ := GetMetadata(out)
	if md.FriendlyName != "VLC" {
		t.Errorf("FriendlyName = %q, want VLC (local wins)", md.FriendlyName)
	}
	if md.Description != "desc" || md.CategoryCode != CategoryMedia {
		t.Errorf("feed did not fill gaps: %+v", md)
	}
	if md.Source != SourceDesktop {
		t.Errorf("Source = %q, want first contributor (desktop)", md.Source)
	}
}
