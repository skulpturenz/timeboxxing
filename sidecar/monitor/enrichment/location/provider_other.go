//go:build !darwin && !windows

package location

// DefaultLocationProvider returns a no-op provider on platforms without a
// supported OS geolocation source (Linux and others). It always reports "no
// fix", so the location enricher contributes only the public IP. A caller can
// inject its own LocationProvider into Enrich if a source is available.
func DefaultLocationProvider() LocationProvider { return nopLocationProvider{} }

type nopLocationProvider struct{}

func (nopLocationProvider) Location() (float64, float64, bool) { return 0, 0, false }

// Permission reports that location is not applicable on this platform, so it is
// omitted from any permissions UI.
func (nopLocationProvider) Permission() (Permission, bool) { return Permission{}, false }
