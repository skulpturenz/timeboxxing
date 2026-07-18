//go:build linux

package platform

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

//go:embed gnome_extension/metadata.json gnome_extension/extension.js
var gnomeExtensionFS embed.FS

const (
	gnomeExtensionUUID = "timeboxxing-focus@skulpture.nz"

	gnomeShellBusName    = "org.gnome.Shell"
	gnomeFocusObjectPath = "/nz/skulpture/Timeboxxing/FocusedWindow"
	gnomeFocusGetMethod  = "nz.skulpture.Timeboxxing.FocusedWindow.Get"

	// The extensions-management API is served under its own well-known bus name
	// (org.gnome.Shell.Extensions), NOT org.gnome.Shell — calling it on the shell
	// name returns "Object does not exist".
	gnomeShellExtensionsName = "org.gnome.Shell.Extensions"
	gnomeExtensionsPath      = "/org/gnome/Shell/Extensions"
	gnomeEnableExtMethod     = "org.gnome.Shell.Extensions.EnableExtension"
)

// gnomeBackend detects the focused window on GNOME/Wayland by calling the
// bundled GNOME Shell extension over D-Bus. Mutter implements no Wayland focus
// protocol, so the extension (running inside the shell) is the only supported
// route. It also yields a PID, enabling /proc-based executable enrichment.
type gnomeBackend struct {
	logger *slog.Logger

	mu   sync.Mutex
	conn *dbus.Conn
	obj  dbus.BusObject
}

func newGnomeBackend(cfg Config, logger *slog.Logger) *gnomeBackend {
	b := &gnomeBackend{logger: logger}

	if err := installGnomeExtension(); err != nil {
		logger.Warn("failed to install bundled GNOME focus extension", "error", err)
	}
	tryEnableGnomeExtension(logger)

	if !b.extensionAvailable() {
		logger.Warn("GNOME focus extension is not active; foreground tracking is disabled until it is enabled",
			"uuid", gnomeExtensionUUID,
			"howto", gnomeEnableInstructions())
	}
	return b
}

func (b *gnomeBackend) poll(now time.Time) (WindowInfo, error) {
	obj, err := b.focusObject()
	if err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, err
	}
	var payloadJSON string
	call := obj.Call(gnomeFocusGetMethod, 0)
	if call.Err != nil {
		// Shell not ready or extension not enabled — drop the connection so the
		// next poll reconnects, and skip this tick.
		b.resetConn()
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, call.Err
	}
	if err := call.Store(&payloadJSON); err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, err
	}

	var payload gnomeFocusPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}, err
	}
	return gnomeWindowInfo(payload, now), nil
}

func (b *gnomeBackend) permissions() []permission.Status {
	return []permission.Status{
		{
			Name:       "GNOME focus extension",
			Granted:    b.extensionAvailable(),
			HowToGrant: gnomeEnableInstructions(),
		},
	}
}

func (b *gnomeBackend) close() {
	b.resetConn()
}

// focusObject returns the D-Bus object for the extension's focus interface,
// (re)connecting to the session bus as needed.
func (b *gnomeBackend) focusObject() (dbus.BusObject, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.obj != nil {
		return b.obj, nil
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	b.conn = conn
	b.obj = conn.Object(gnomeShellBusName, dbus.ObjectPath(gnomeFocusObjectPath))
	return b.obj, nil
}

func (b *gnomeBackend) resetConn() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		_ = b.conn.Close()
	}
	b.conn = nil
	b.obj = nil
}

// extensionAvailable reports whether the extension's D-Bus method responds.
func (b *gnomeBackend) extensionAvailable() bool {
	obj, err := b.focusObject()
	if err != nil {
		return false
	}
	var s string
	call := obj.Call(gnomeFocusGetMethod, 0)
	if call.Err != nil {
		b.resetConn()
		return false
	}
	return call.Store(&s) == nil
}

type gnomeFocusPayload struct {
	WMClass string `json:"wm_class"`
	Title   string `json:"title"`
	PID     int32  `json:"pid"`
}

func gnomeWindowInfo(payload gnomeFocusPayload, now time.Time) WindowInfo {
	// No focused window (the extension returns {}) — report no foreground so the
	// session manager keeps the current session rather than inventing "Unknown app".
	if payload.WMClass == "" && payload.Title == "" && payload.PID <= 0 {
		return WindowInfo{Timestamp: now, TitleSource: TitleSourceNone}
	}
	appName, appPath := "", ""
	if payload.PID > 0 {
		appName = procComm(uint32(payload.PID))
		appPath = procExe(uint32(payload.PID))
	}
	identity := normalizeLinuxAppIdentity(appName, appPath, payload.WMClass, payload.Title)
	return WindowInfo{
		AppName:       identity.AppName,
		AppIdentifier: identity.AppIdentifier,
		AppPath:       identity.AppPath,
		PID:           payload.PID,
		WindowTitle:   payload.Title,
		TitleSource:   TitleSourceWindowAPI,
		Timestamp:     now,
	}
}

// installGnomeExtension writes the bundled extension into the user's GNOME
// extensions directory if not already present or out of date.
func installGnomeExtension() error {
	dir, err := gnomeExtensionInstallDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(gnomeExtensionFS, "gnome_extension")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := gnomeExtensionFS.ReadFile("gnome_extension/" + entry.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func gnomeExtensionInstallDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "gnome-shell", "extensions", gnomeExtensionUUID), nil
}

// tryEnableGnomeExtension best-effort enables the extension via GNOME Shell's
// D-Bus API. A freshly-installed extension still needs a shell reload
// (logout/login on Wayland) before it loads, so this is not sufficient on its
// own, but it avoids a manual step once the shell has picked the files up.
func tryEnableGnomeExtension(logger *slog.Logger) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return
	}
	defer conn.Close()

	obj := conn.Object(gnomeShellExtensionsName, dbus.ObjectPath(gnomeExtensionsPath))
	call := obj.Call(gnomeEnableExtMethod, 0, gnomeExtensionUUID)
	if call.Err != nil {
		logger.Debug("could not auto-enable GNOME focus extension (enable it manually and re-log)", "error", call.Err)
		return
	}
	var enabled bool
	_ = call.Store(&enabled)
	if enabled {
		logger.Info("enabled GNOME focus extension", "uuid", gnomeExtensionUUID)
	} else {
		// The shell only knows a freshly-installed extension after a reload
		// (logout/login on Wayland); the next launch after that enables it.
		logger.Info("GNOME focus extension installed but not yet enabled; log out and back in, then relaunch",
			"uuid", gnomeExtensionUUID)
	}
}

func gnomeEnableInstructions() string {
	return "run `gnome-extensions enable " + gnomeExtensionUUID + "` (or use the Extensions app), then log out and back in"
}
