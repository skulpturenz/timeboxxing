package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

type fakeLocation struct {
	lat, lon float64
	ok       bool
	perm     permission.Status
	permOK   bool
}

func (f fakeLocation) Location() (float64, float64, bool)    { return f.lat, f.lon, f.ok }
func (f fakeLocation) Permission() (permission.Status, bool) { return f.perm, f.permOK }
func (f fakeLocation) RequestPermission(context.Context)     {}

type fakeIP struct{ ip *string }

func (f fakeIP) PublicIP() *string { return f.ip }

func strptr(s string) *string { return &s }

func emptyFP() monitor.ForegroundProcess {
	return monitor.ForegroundProcess{Enrichments: map[string]any{}}
}

func TestEnrich_LocationAndIP(t *testing.T) {
	loc := fakeLocation{lat: 40.71, lon: -74.0, ok: true}
	ip := fakeIP{ip: strptr("203.0.113.7")}

	out, ok := Enrich(loc, ip)(context.Background(), emptyFP())
	require.True(t, ok, "expected enrichment")
	env, ok := Get(out)
	require.True(t, ok, "environment not stored in bag")

	require.NotNil(t, env.Latitude)
	assert.Equal(t, 40.71, *env.Latitude)
	require.NotNil(t, env.Longitude)
	assert.Equal(t, -74.0, *env.Longitude)
	require.NotNil(t, env.PublicIP)
	assert.Equal(t, "203.0.113.7", *env.PublicIP)
}

func TestEnrich_IPOnlyWhenNoFix(t *testing.T) {
	out, ok := Enrich(fakeLocation{ok: false}, fakeIP{ip: strptr("198.51.100.9")})(context.Background(), emptyFP())
	require.True(t, ok, "expected enrichment from public IP alone")
	env, _ := Get(out)
	assert.Nil(t, env.Latitude, "expected no location latitude")
	assert.Nil(t, env.Longitude, "expected no location longitude")
	require.NotNil(t, env.PublicIP)
	assert.Equal(t, "198.51.100.9", *env.PublicIP)
}

func TestEnrich_NothingToContribute(t *testing.T) {
	_, ok := Enrich(fakeLocation{ok: false}, fakeIP{ip: nil})(context.Background(), emptyFP())
	assert.False(t, ok, "expected no enrichment when neither provider has data")
}

func TestEnrich_NilProviders(t *testing.T) {
	_, ok := Enrich(nil, nil)(context.Background(), emptyFP())
	assert.False(t, ok, "expected no-op with nil providers")
}

func TestRequestable(t *testing.T) {
	// Applicable provider surfaces a requestable permission reflecting its status.
	granted := fakeLocation{perm: permission.Status{Name: "Location Services", Granted: true}, permOK: true}
	perm, ok := Requestable(granted)
	require.True(t, ok)
	assert.Equal(t, "Location Services", perm.Name())
	assert.True(t, perm.Granted())

	// Unsupported platform (applicable=false) is omitted.
	_, ok = Requestable(fakeLocation{permOK: false})
	assert.False(t, ok, "want not-ok for unsupported provider")

	// Nil provider is safe.
	_, ok = Requestable(nil)
	assert.False(t, ok, "Requestable(nil) should be not-ok")
}

func TestPublicIPProvider_MemoizesLookup(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("192.0.2.44\n"))
	}))
	defer srv.Close()

	p := NewPublicIPProvider()
	p.endpoint = srv.URL

	ip := p.PublicIP() // first call performs the (blocking) lookup
	require.NotNil(t, ip)
	require.Equal(t, "192.0.2.44", *ip)

	// Returned pointer must be a copy — mutating it must not corrupt the cache.
	*ip = "mutated"
	again := p.PublicIP()
	require.NotNil(t, again)
	assert.Equal(t, "192.0.2.44", *again, "cache was mutated through returned pointer")

	// Subsequent calls must hit the memo, not the network.
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "server should be hit once (memoized)")
}

func TestParsePublicIP(t *testing.T) {
	cases := map[string]string{
		"203.0.113.1\n":      "203.0.113.1",
		"  2001:db8::1  ":    "2001:db8::1",
		"not-an-ip":          "",
		"":                   "",
		"<html>error</html>": "",
	}
	for in, want := range cases {
		assert.Equalf(t, want, parsePublicIP(in), "parsePublicIP(%q)", in)
	}
}
