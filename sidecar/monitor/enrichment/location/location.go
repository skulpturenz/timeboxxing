// Package location provides an enricher that annotates a foreground process with
// ambient environment context — the machine's current geographic location (when
// the OS exposes it) and its public IP address. Unlike the app-metadata and
// browser enrichers, this data is not derived from the process itself; it is read
// from cached, non-blocking providers so the poll loop is never stalled on GPS or
// network I/O. See the "Location + public IP" plan for the capture approach.
package location

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// Key is the Enrichments bag key under which the Environment payload is stored.
const Key = "location"

// Environment is the ambient context for a foreground observation. Every field
// is nullable: a nil Latitude/Longitude means "no location fix" (distinct from a
// literal 0,0), and a nil PublicIP means "not yet resolved / offline".
type Environment struct {
	Latitude  *float64
	Longitude *float64
	PublicIP  *string
}

// LocationProvider yields the latest cached geographic fix and reports the OS
// location permission. Location() must be non-blocking; ok is false when no fix
// is available (no permission, no fix yet, or an unsupported platform).
type LocationProvider interface {
	Location() (lat float64, lon float64, ok bool)
	// Permission reports the OS location-permission status. applicable is false
	// on platforms where location is unsupported (e.g. Linux), so callers can
	// omit it from a permissions UI.
	Permission() (status permission.Status, applicable bool)
	// RequestPermission triggers the OS location-authorization prompt. It is a
	// no-op on platforms where location is unsupported. Requesting is deferred to
	// this call (rather than provider construction) so the caller controls when
	// the prompt fires.
	RequestPermission(ctx context.Context)
}

// PublicIPProvider yields the cached public IP, or nil when unknown. Reads must
// be non-blocking.
type PublicIPProvider interface {
	PublicIP() *string
}

// Enrich returns an enricher that tags the process with location and public IP
// read from the given providers. Either provider may be nil to disable that
// dimension. It is NOT memoized: the values are ambient and read straight from
// each provider's cache on every poll. Returns (fp, false) when neither provider
// has anything to contribute.
func Enrich(location LocationProvider, publicIP PublicIPProvider) enrichment.Enricher {
	return func(_ context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		env := Environment{}
		changed := false

		if location != nil {
			if lat, lon, ok := location.Location(); ok {
				env.Latitude = new(lat)
				env.Longitude = new(lon)
				changed = true
			}
		}
		if publicIP != nil {
			if ip := publicIP.PublicIP(); ip != nil {
				env.PublicIP = ip
				changed = true
			}
		}

		if !changed {
			return fp, false
		}

		updated := fp
		// An idle process reaches here (public IP is machine-wide) with a nil map.
		if updated.Enrichments == nil {
			updated.Enrichments = map[string]any{}
		}
		updated.Enrichments[Key] = &env
		return updated, true
	}
}

// Requestable adapts a LocationProvider into a permission.Permission so a caller
// (e.g. monitor.New) can request the OS location authorization uniformly with
// other monitor permissions. Returns ok=false when location is unsupported on
// this platform, so it is omitted from the permission set.
func Requestable(provider LocationProvider) (permission.Permission, bool) {
	if provider == nil {
		return nil, false
	}
	if _, applicable := provider.Permission(); !applicable {
		return nil, false
	}
	return providerPermission{provider: provider}, true
}

// providerPermission is the permission.Permission view of a LocationProvider. It
// re-reads the live status on each accessor and delegates Request to the provider.
type providerPermission struct {
	provider LocationProvider
}

func (p providerPermission) Name() string { perm, _ := p.provider.Permission(); return perm.Name }
func (p providerPermission) HowToGrant() string {
	perm, _ := p.provider.Permission()
	return perm.HowToGrant
}
func (p providerPermission) Granted() bool { perm, _ := p.provider.Permission(); return perm.Granted }

func (p providerPermission) Request(ctx context.Context) { p.provider.RequestPermission(ctx) }

// Get returns the Environment stored on the process, if any.
func Get(fp monitor.ForegroundProcess) (Environment, bool) {
	if fp.Enrichments == nil {
		return Environment{}, false
	}
	value, ok := fp.Enrichments[Key]
	if !ok {
		return Environment{}, false
	}
	env, ok := value.(*Environment)
	if !ok || env == nil {
		return Environment{}, false
	}
	return *env, true
}
