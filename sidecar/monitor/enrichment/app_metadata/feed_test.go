package appmetadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment"
)

// TestFlathub_Live hits the real Flathub endpoint to confirm the response still
// decodes into our struct. Opt-in (network): set TBX_LIVE_FEEDS=1 to run.
func TestFlathub_Live(t *testing.T) {
	if os.Getenv("TBX_LIVE_FEEDS") == "" {
		t.Skip("set TBX_LIVE_FEEDS=1 to run live feed tests")
	}
	out, ok := FlathubEnricher(context.Background(), fpWithID("org.videolan.VLC"))
	require.True(t, ok, "expected live Flathub enrichment for VLC")
	md, _ := GetMetadata(out)
	require.NotEmpty(t, md.FriendlyName, "live response under-populated")
	require.NotEqual(t, CategoryUnknown, md.Category, "live response under-populated")
	t.Logf("live VLC => name=%q category=%q desc.len=%d", md.FriendlyName, md.Category.Label(), len(md.Description))
}

func ptr(s string) *string { return &s }

func fpWithID(id string) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{
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

	out, ok := FlathubEnricher(context.Background(), fpWithID("org.videolan.VLC"))
	require.True(t, ok, "expected Flathub to enrich")
	md, _ := GetMetadata(out)
	assert.Equal(t, "VLC", md.FriendlyName)
	assert.Equal(t, CategoryMedia, md.Category)
	assert.Equal(t, SourceFlathub, md.Source)
	assert.NotEmpty(t, md.Description, "expected a description from the summary")
}

func TestFlathub_SkipsWhenLocalComplete(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(vlcAppstreamJSON))
	}))
	defer srv.Close()
	old := flathubBaseURL
	flathubBaseURL = srv.URL + "/"
	defer func() { flathubBaseURL = old }()

	// Local enrichment already filled everything -> feed must not fire.
	fp := fpWithID("org.videolan.VLC")
	fp, _ = setMetadata(fp, Metadata{
		FriendlyName: "VLC", Description: "d", Category: CategoryMedia,
		IconPath: "/tmp/x.png", Source: SourceDesktop,
	})

	_, ok := FlathubEnricher(context.Background(), fp)
	assert.False(t, ok, "expected no-op when metadata already complete")
	assert.Equal(t, int32(0), hits.Load(), "feed must not be hit when metadata is complete")
}

func TestFlathub_Disabled(t *testing.T) {
	// An unconfigured (empty) base URL disables the feed.
	old := flathubBaseURL
	flathubBaseURL = ""
	defer func() { flathubBaseURL = old }()

	_, ok := FlathubEnricher(context.Background(), fpWithID("org.videolan.VLC"))
	assert.False(t, ok, "disabled Flathub should be a no-op")
}

func TestFlathub_IgnoresNonReverseDNS(t *testing.T) {
	// "chrome" (Windows-style exe id) is not a flatpak app id.
	_, ok := FlathubEnricher(context.Background(), fpWithID("chrome"))
	assert.False(t, ok, "expected no-op for non reverse-DNS identifier")
}

func TestMemoized_MemoizesByIdentity(t *testing.T) {
	var calls int32
	inner := func(_ context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		atomic.AddInt32(&calls, 1)
		return setMetadata(fp, Metadata{FriendlyName: "X", Source: SourceBundle})
	}
	enricher := Memoized(inner)

	for i := 0; i < 3; i++ {
		out, ok := enricher(context.Background(), fpWithID("com.example.app"))
		require.True(t, ok, "expected enrichment")
		md, _ := GetMetadata(out)
		assert.Equal(t, "X", md.FriendlyName)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "inner should be called once (memoized)")
}

func TestFold_MergesLocalThenFeed(t *testing.T) {
	local := func(_ context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		return setMetadata(fp, Metadata{FriendlyName: "VLC", IconPath: "/i.png", Source: SourceDesktop})
	}
	feed := func(_ context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		// Feed only fills category/description; must not overwrite the name.
		return setMetadata(fp, Metadata{FriendlyName: "WRONG", Description: "desc", Category: CategoryMedia, Source: SourceFlathub})
	}

	out, ok := enrichment.Pipe(local, feed)(context.Background(), fpWithID("org.videolan.VLC"))
	require.True(t, ok, "expected combined enrichment")
	md, _ := GetMetadata(out)
	assert.Equal(t, "VLC", md.FriendlyName, "local should win")
	assert.Equal(t, "desc", md.Description, "feed should fill description gap")
	assert.Equal(t, CategoryMedia, md.Category, "feed should fill category gap")
	assert.Equal(t, SourceDesktop, md.Source, "Source is the first contributor (desktop)")
}
