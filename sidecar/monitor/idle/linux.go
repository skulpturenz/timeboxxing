//go:build linux

package idle

import (
	"context"
	"os"
	"strings"
)

// New returns the Linux IdleDetector, selecting an implementation for the
// current session: ext-idle-notify-v1 on Wayland, the X11 MIT-SCREEN-SAVER
// extension otherwise. Falls back to a no-op detector when neither is available
// so the rest of the app keeps working.
func New(ctx context.Context) (IdleDetector, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if isWaylandSession() {
		if d := newWaylandIdleDetector(); d != nil {
			return d, nil
		}
		// Wayland session without ext-idle-notify: XWayland's screensaver
		// extension may still work, otherwise degrade to no-op below.
	}

	if d := newX11IdleDetector(); d != nil {
		return d, nil
	}
	return Nop(), nil
}

// isWaylandSession reports whether the process runs under a Wayland compositor.
func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")), "wayland")
}
