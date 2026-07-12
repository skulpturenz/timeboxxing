//go:build linux

package wlr

// The scanner-generated binding references core Wayland types by their bare
// names (BaseProxy, Context, Event, Seat, Output, Surface). These aliases wire
// them to github.com/neurlang/wayland/wl so the generated file compiles as a
// standalone package.

import "github.com/neurlang/wayland/wl"

type (
	BaseProxy = wl.BaseProxy
	Context   = wl.Context
	Event     = wl.Event
	Seat      = wl.Seat
	Output    = wl.Output
	Surface   = wl.Surface
)

// SafeCast wraps wl.SafeCast so the generated binding can resolve it locally.
func SafeCast[T any](p wl.Proxy) T {
	return wl.SafeCast[T](p)
}
