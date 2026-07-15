//go:build linux

package platform

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// linuxBackend is the internal abstraction each Linux windowing backend
// (X11, wlr, KDE plasma, GNOME) implements. The exported linuxTracker wraps the
// selected backend and applies the shared ctx/timestamp/finalize handling.
type linuxBackend interface {
	// poll returns the current foreground window. Event-driven Wayland
	// backends return a cached snapshot; the X11 backend queries synchronously.
	poll(now time.Time) (WindowInfo, error)
	permissions() []PermissionStatus
	close()
}

type linuxTracker struct {
	cfg     Config
	backend linuxBackend
}

// New returns the Linux Tracker implementation, selecting a windowing backend
// appropriate for the current session (X11 or a supported Wayland compositor).
func New(ctx context.Context, cfg Config) (Tracker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	backend := selectLinuxBackend(ctx, cfg, logger)
	return &linuxTracker{cfg: cfg, backend: backend}, nil
}

func (t *linuxTracker) Poll(ctx context.Context) (WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return WindowInfo{}, err
	}
	now := time.Now()
	info, err := t.backend.poll(now)
	if err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, err
	}
	// Location is not available on Linux desktops, and public IP is now attached
	// by the location enricher (sidecar/monitor/encrichment/location), so there is
	// no environment context to add here.
	info, _ = FinalizeWindowInfo(info)
	return info, nil
}

func (t *linuxTracker) Permissions() []PermissionStatus {
	return t.backend.permissions()
}

// selectLinuxBackend picks the best available backend for the current session.
// Wayland sessions are tried first with the compositor's toplevel protocols;
// X11 (including XWayland) uses EWMH; anything else degrades gracefully.
func selectLinuxBackend(ctx context.Context, cfg Config, logger *slog.Logger) linuxBackend {
	if isWaylandSession() {
		if b := newWaylandForegroundBackend(cfg, logger); b != nil {
			return b
		}
		if desktopIsGNOME() {
			logger.InfoContext(ctx, "using GNOME Shell extension backend for Wayland foreground tracking")
			return newGnomeBackend(cfg, logger)
		}
		// Some GNOME/obscure compositors expose no focus protocol. Try XWayland
		// (which only sees X11 clients) before giving up entirely.
		if os.Getenv("DISPLAY") != "" {
			if b := newX11Backend(); b != nil {
				logger.WarnContext(ctx, "no native Wayland focus protocol available; falling back to XWayland (native Wayland windows will not be tracked)")
				return b
			}
		}
		logger.WarnContext(ctx, "no supported Wayland window protocol and no X11 fallback; foreground tracking disabled",
			"desktop", os.Getenv("XDG_CURRENT_DESKTOP"))
		return newDegradedBackend("this Wayland compositor exposes no supported active-window protocol")
	}

	if b := newX11Backend(); b != nil {
		return b
	}
	logger.WarnContext(ctx, "no X11 display reachable; foreground tracking disabled")
	return newDegradedBackend("no X11 display reachable (set DISPLAY, or start a supported Wayland compositor)")
}

// isWaylandSession reports whether the process is running under a Wayland
// compositor, preferring native Wayland over XWayland when both are present.
func isWaylandSession() bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")), "wayland")
}

// desktopIsGNOME reports whether the current desktop is GNOME (or a GNOME-based
// shell), which requires the companion Shell extension for focus tracking.
func desktopIsGNOME() bool {
	for _, key := range []string{"XDG_CURRENT_DESKTOP", "XDG_SESSION_DESKTOP", "DESKTOP_SESSION"} {
		if strings.Contains(strings.ToLower(os.Getenv(key)), "gnome") {
			return true
		}
	}
	return false
}

// degradedBackend is used when no window protocol is available. It never
// reports a foreground window but keeps the app running and surfaces the reason
// through Permissions().
type degradedBackend struct {
	reason string
}

func newDegradedBackend(reason string) *degradedBackend { return &degradedBackend{reason: reason} }

func (d *degradedBackend) poll(now time.Time) (WindowInfo, error) {
	return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, nil
}

func (d *degradedBackend) permissions() []PermissionStatus {
	return []PermissionStatus{
		{Name: "Active-window protocol", Granted: false, HowToGrant: d.reason},
	}
}

func (d *degradedBackend) close() {}

// --- shared Linux app-identity helpers (used by every backend) ---

type linuxAppIdentity struct {
	AppName       string
	AppIdentifier string
	AppPath       string
}

func normalizeLinuxAppIdentity(appName string, appPath string, windowClass string, windowTitle string) linuxAppIdentity {
	appName = strings.TrimSpace(appName)
	appPath = strings.TrimSpace(appPath)
	windowClass = strings.TrimSpace(windowClass)
	windowTitle = strings.TrimSpace(windowTitle)
	if runtimeName, ok := linuxRuntimeExecutableName(appName, appPath); ok {
		return linuxAppIdentity{
			AppName:       runtimeName,
			AppIdentifier: runtimeName,
			AppPath:       appPath,
		}
	}
	appName = firstNonEmpty(appName, displayNameFromPath(appPath), windowClass, windowTitle)
	return linuxAppIdentity{
		AppName:       appName,
		AppIdentifier: linuxAppIdentifier(appName, appPath, windowClass),
		AppPath:       appPath,
	}
}

func linuxRuntimeExecutableName(appName string, appPath string) (string, bool) {
	names := []string{
		strings.TrimSpace(appName),
		filepath.Base(strings.TrimSpace(appPath)),
	}
	for _, name := range names {
		switch strings.ToLower(name) {
		case "java", "javaw":
			return strings.ToLower(name), true
		}
	}
	return "", false
}

func linuxAppIdentifier(appName string, appPath string, windowClass string) string {
	if class := strings.TrimSpace(windowClass); class != "" {
		return class
	}
	if appPath != "" {
		return filepath.Base(appPath)
	}
	return appName
}

func parseWMClass(value []byte) string {
	raw := strings.TrimRight(string(value), "\x00")
	parts := strings.Split(raw, "\x00")
	for i := len(parts) - 1; i >= 0; i-- {
		if part := strings.TrimSpace(parts[i]); part != "" {
			return part
		}
	}
	return ""
}

// procComm reads the process name from /proc/<pid>/comm.
func procComm(pid uint32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// procExe reads the executable path from /proc/<pid>/exe symlink.
func procExe(pid uint32) string {
	link, err := filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return link
}

// sanitiseLatin1 converts a Latin-1 byte slice to a printable UTF-8 string.
func sanitiseLatin1(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c >= 0x20 && c != 0x7f {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
