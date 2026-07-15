package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

type fakeLocation struct {
	lat, lon float64
	ok       bool
	perm     Permission
	permOK   bool
}

func (f fakeLocation) Location() (float64, float64, bool) { return f.lat, f.lon, f.ok }
func (f fakeLocation) Permission() (Permission, bool)     { return f.perm, f.permOK }

type fakeIP struct{ ip *string }

func (f fakeIP) PublicIP() *string { return f.ip }

func strptr(s string) *string { return &s }

func emptyFP() sessionnew.ForegroundProcess {
	return sessionnew.ForegroundProcess{Enrichments: map[string]any{}}
}

func TestEnrich_LocationAndIP(t *testing.T) {
	loc := fakeLocation{lat: 40.71, lon: -74.0, ok: true}
	ip := fakeIP{ip: strptr("203.0.113.7")}

	out, ok := Enrich(loc, ip)(context.Background(), emptyFP())
	if !ok {
		t.Fatal("expected enrichment")
	}
	env, ok := Get(out)
	if !ok {
		t.Fatal("environment not stored in bag")
	}
	if env.Latitude == nil || *env.Latitude != 40.71 {
		t.Errorf("Latitude = %v, want 40.71", env.Latitude)
	}
	if env.Longitude == nil || *env.Longitude != -74.0 {
		t.Errorf("Longitude = %v, want -74.0", env.Longitude)
	}
	if env.PublicIP == nil || *env.PublicIP != "203.0.113.7" {
		t.Errorf("PublicIP = %v", env.PublicIP)
	}
}

func TestEnrich_IPOnlyWhenNoFix(t *testing.T) {
	out, ok := Enrich(fakeLocation{ok: false}, fakeIP{ip: strptr("198.51.100.9")})(context.Background(), emptyFP())
	if !ok {
		t.Fatal("expected enrichment from public IP alone")
	}
	env, _ := Get(out)
	if env.Latitude != nil || env.Longitude != nil {
		t.Errorf("expected no location, got lat=%v lon=%v", env.Latitude, env.Longitude)
	}
	if env.PublicIP == nil || *env.PublicIP != "198.51.100.9" {
		t.Errorf("PublicIP = %v", env.PublicIP)
	}
}

func TestEnrich_NothingToContribute(t *testing.T) {
	_, ok := Enrich(fakeLocation{ok: false}, fakeIP{ip: nil})(context.Background(), emptyFP())
	if ok {
		t.Error("expected no enrichment when neither provider has data")
	}
}

func TestEnrich_NilProviders(t *testing.T) {
	_, ok := Enrich(nil, nil)(context.Background(), emptyFP())
	if ok {
		t.Error("expected no-op with nil providers")
	}
}

func TestPermissions(t *testing.T) {
	// Applicable provider surfaces its permission.
	granted := fakeLocation{perm: Permission{Name: "Location Services", Granted: true}, permOK: true}
	perms := Permissions(granted)
	if len(perms) != 1 || perms[0].Name != "Location Services" || !perms[0].Granted {
		t.Errorf("Permissions = %+v, want one granted Location Services", perms)
	}

	// Unsupported platform (applicable=false) yields no permissions.
	if got := Permissions(fakeLocation{permOK: false}); len(got) != 0 {
		t.Errorf("Permissions = %+v, want empty for unsupported provider", got)
	}

	// Nil provider is safe.
	if got := Permissions(nil); got != nil {
		t.Errorf("Permissions(nil) = %+v, want nil", got)
	}
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
	if ip == nil || *ip != "192.0.2.44" {
		t.Fatalf("PublicIP = %v, want 192.0.2.44", ip)
	}

	// Returned pointer must be a copy — mutating it must not corrupt the cache.
	*ip = "mutated"
	if again := p.PublicIP(); again == nil || *again != "192.0.2.44" {
		t.Errorf("cache was mutated through returned pointer: %v", again)
	}

	// Subsequent calls must hit the memo, not the network.
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("server hit %d times, want 1 (memoized)", got)
	}
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
		if got := parsePublicIP(in); got != want {
			t.Errorf("parsePublicIP(%q) = %q, want %q", in, got, want)
		}
	}
}
