package appmetadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// TestFlathub_Live hits the real Flathub endpoint to confirm the response still
// decodes into our struct. Opt-in (network): set TBX_LIVE_FEEDS=1 to run.
func TestFlathub_Live(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flathub enricher only runs on Linux")
	}
	if os.Getenv("TBX_LIVE_FEEDS") == "" {
		t.Skip("set TBX_LIVE_FEEDS=1 to run live feed tests")
	}
	out, ok := FlathubEnricher(context.Background(), fpWithID("org.videolan.VLC"))
	require.True(t, ok, "expected live Flathub enrichment for VLC")
	md, _ := out.Enrichments[KeyMetadata].(*Metadata)
	require.NotEmpty(t, md.FriendlyName, "live response under-populated")
	require.NotEqual(t, CategoryUnknown, md.Category, "live response under-populated")
	t.Logf("live VLC => name=%q category=%q desc.len=%d", md.FriendlyName, md.Category.Label(), len(md.Description))
}

func fpWithID(id string) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{
		AppIdentifier: new(id),
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
	if runtime.GOOS != "linux" {
		t.Skip("Flathub enricher only runs on Linux")
	}
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
	md, _ := out.Enrichments[KeyMetadata].(*Metadata)
	assert.Equal(t, "VLC", md.FriendlyName)
	assert.Equal(t, CategoryMedia, md.Category)
	assert.Equal(t, SourceFlathub, md.Source)
	assert.NotEmpty(t, md.Description, "expected a description from the summary")
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
		updated := fp
		updated.Enrichments[KeyMetadata] = &Metadata{FriendlyName: "X", Source: SourceBundle}
		return updated, true
	}
	enricher := Memoized(inner)

	for i := 0; i < 3; i++ {
		out, ok := enricher(context.Background(), fpWithID("com.example.app"))
		require.True(t, ok, "expected enrichment")
		md, _ := out.Enrichments[KeyMetadata].(*Metadata)
		assert.Equal(t, "X", md.FriendlyName)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "inner should be called once (memoized)")
}

func TestGetJSON_RetriesOn5xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"ok"}`))
	}))
	defer srv.Close()

	var out struct {
		Name string `json:"name"`
	}
	ok, err := getJSON(context.Background(), srv.URL, &out)
	require.NoError(t, err)
	assert.True(t, ok, "expected success after retrying the transient 500")
	assert.Equal(t, "ok", out.Name)
	assert.Equal(t, int32(2), hits.Load(), "expected one retry after the 500 (2 hits total)")
}

func TestGetJSON_DoesNotRetryOn404(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var out struct{}
	ok, err := getJSON(context.Background(), srv.URL, &out)
	require.NoError(t, err)
	assert.False(t, ok, "404 is a not-found, not an error")
	assert.Equal(t, int32(1), hits.Load(), "a 404 must not be retried")
}
