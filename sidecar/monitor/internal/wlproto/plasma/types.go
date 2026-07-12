//go:build linux

package plasma

// Aliases wiring the scanner-generated binding's bare core-type references to
// github.com/neurlang/wayland/wl.

import "github.com/neurlang/wayland/wl"

type (
	BaseProxy = wl.BaseProxy
	Context   = wl.Context
	Event     = wl.Event
	Output    = wl.Output
	Surface   = wl.Surface
)

// SafeCast wraps wl.SafeCast so the generated binding can resolve it locally.
func SafeCast[T any](p wl.Proxy) T {
	return wl.SafeCast[T](p)
}
