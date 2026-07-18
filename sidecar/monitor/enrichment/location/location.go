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
	Permission() (perm Permission, applicable bool)
}

// PublicIPProvider yields the cached public IP, or nil when unknown. Reads must
// be non-blocking.
type PublicIPProvider interface {
	PublicIP() *string
}

// Permission describes an OS permission the location enricher relies on, in the
// same shape as the platform tracker's PermissionStatus so a UI can present them
// together.
type Permission struct {
	Name       string
	Granted    bool
	HowToGrant string
}

// Permissions returns the OS permissions the location provider needs, suitable
// for surfacing in a settings UI. Empty when the provider is nil or location is
// unsupported on this platform.
func Permissions(provider LocationProvider) []Permission {
	if provider == nil {
		return nil
	}
	if perm, ok := provider.Permission(); ok {
		return []Permission{perm}
	}
	return nil
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
				latitude, longitude := lat, lon
				env.Latitude = &latitude
				env.Longitude = &longitude
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
		return setEnv(fp, env), true
	}
}

// Default composes Enrich with the platform's default location provider and a
// fresh public-IP provider. This is the enricher most callers want.
func Default() enrichment.Enricher {
	return Enrich(DefaultLocationProvider(), NewPublicIPProvider())
}

// Get returns the Environment stored on the process, if any.
func Get(fp monitor.ForegroundProcess) (Environment, bool) {
	if fp.Enrichments == nil {
		return Environment{}, false
	}
	value, ok := fp.Enrichments[Key]
	if !ok {
		return Environment{}, false
	}
	env, ok := value.(Environment)
	return env, ok
}

// setEnv writes env into a cloned Enrichments bag (the input is treated as
// immutable) and returns the updated process.
func setEnv(fp monitor.ForegroundProcess, env Environment) monitor.ForegroundProcess {
	bag := make(map[string]any, len(fp.Enrichments)+1)
	for k, v := range fp.Enrichments {
		bag[k] = v
	}
	bag[Key] = env
	fp.Enrichments = bag
	return fp
}
