//go:build linux

package platform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseWMClassPrefersClassName(t *testing.T) {
	got := parseWMClass([]byte("code\x00Code\x00"))
	require.Equal(t, "Code", got, "expected class name Code")
}

func TestParseWMClassFallsBackToInstanceName(t *testing.T) {
	got := parseWMClass([]byte("firefox\x00\x00"))
	require.Equal(t, "firefox", got, "expected instance name firefox")
}

func TestLinuxAppIdentifierPrefersWMClass(t *testing.T) {
	got := linuxAppIdentifier("code", "/usr/share/code/code", "Code")
	require.Equal(t, "Code", got, "expected WM_CLASS identifier")
}

func TestLinuxAppIdentifierFallsBackToExecutablePath(t *testing.T) {
	got := linuxAppIdentifier("code", "/usr/share/code/code", "")
	require.Equal(t, "code", got, "expected executable basename")
}

func TestNormalizeLinuxAppIdentityPrefersJavaRuntime(t *testing.T) {
	got := normalizeLinuxAppIdentity("java", "/usr/lib/jvm/temurin/bin/java", "jetbrains-idea", "Project")

	require.Equal(t, "java", got.AppName)
	require.Equal(t, "java", got.AppIdentifier)
	require.Equal(t, "/usr/lib/jvm/temurin/bin/java", got.AppPath)
}

func TestNormalizeLinuxAppIdentityPreservesWindowClassForNormalApps(t *testing.T) {
	got := normalizeLinuxAppIdentity("code", "/usr/share/code/code", "Code", "main.go")

	require.Equal(t, "code", got.AppName)
	require.Equal(t, "Code", got.AppIdentifier, "expected WM_CLASS identifier")
	require.Equal(t, "/usr/share/code/code", got.AppPath)
}

func TestNormalizeLinuxAppIdentityFallsBackToWindowClass(t *testing.T) {
	got := normalizeLinuxAppIdentity("", "", "Code", "main.go")

	require.Equal(t, "Code", got.AppName, "expected WM_CLASS app name")
	require.Equal(t, "Code", got.AppIdentifier, "expected WM_CLASS identifier")
}

func TestNormalizeLinuxAppIdentityFallsBackToWindowTitle(t *testing.T) {
	got := normalizeLinuxAppIdentity("", "", "", "Personal — Instagram")

	require.Equal(t, "Personal — Instagram", got.AppName, "expected window title app name")
	require.Equal(t, "Personal — Instagram", got.AppIdentifier, "expected title identifier")
}

func TestIsWaylandSession(t *testing.T) {
	cases := []struct {
		name        string
		waylandDisp string
		sessionType string
		want        bool
	}{
		{"wayland display set", "wayland-0", "", true},
		{"session type wayland", "", "wayland", true},
		{"session type wayland uppercase", "", "Wayland", true},
		{"x11 session", "", "x11", false},
		{"nothing set", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("WAYLAND_DISPLAY", tc.waylandDisp)
			t.Setenv("XDG_SESSION_TYPE", tc.sessionType)
			require.Equal(t, tc.want, isWaylandSession())
		})
	}
}

func TestDesktopIsGNOME(t *testing.T) {
	cases := []struct {
		name    string
		current string
		want    bool
	}{
		{"plain GNOME", "GNOME", true},
		{"ubuntu GNOME", "ubuntu:GNOME", true},
		{"lowercase gnome", "gnome", true},
		{"KDE", "KDE", false},
		{"sway", "sway", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CURRENT_DESKTOP", tc.current)
			t.Setenv("XDG_SESSION_DESKTOP", "")
			t.Setenv("DESKTOP_SESSION", "")
			require.Equal(t, tc.want, desktopIsGNOME())
		})
	}
}

func TestWaylandWindowInfoDerivesAppNameFromReverseDNS(t *testing.T) {
	got := waylandWindowInfo("org.mozilla.firefox", "Reddit — Mozilla Firefox")
	require.Equal(t, "firefox", got.AppName)
	require.Equal(t, "org.mozilla.firefox", got.AppIdentifier, "expected app_id identifier")
	require.Equal(t, "Reddit — Mozilla Firefox", got.WindowTitle, "expected window title preserved")
	require.Zerof(t, got.PID, "expected no PID from wlr protocol, got pid=%d", got.PID)
	require.Emptyf(t, got.AppPath, "expected no path from wlr protocol, got path=%q", got.AppPath)
}

func TestWaylandWindowInfoSimpleAppId(t *testing.T) {
	got := waylandWindowInfo("code", "main.go - timeboxxing")
	require.Equal(t, "code", got.AppName)
	require.Equal(t, "code", got.AppIdentifier)
}

func TestWaylandWindowInfoFallsBackToTitle(t *testing.T) {
	got := waylandWindowInfo("", "Some Window")
	require.Equal(t, "Some Window", got.AppName, "expected title fallback app name")
	require.Empty(t, got.AppIdentifier, "expected empty identifier")
}

func TestGnomeWindowInfoParsesPayload(t *testing.T) {
	got := gnomeWindowInfo(gnomeFocusPayload{WMClass: "Navigator", Title: "Reddit - Firefox", PID: 0}, time.Now())
	require.Equal(t, "Navigator", got.AppName, "expected app name from wm_class")
	require.Equal(t, "Navigator", got.AppIdentifier, "expected identifier from wm_class")
	require.Equal(t, "Reddit - Firefox", got.WindowTitle)
}

func TestGnomeWindowInfoEmptyFocusReportsNone(t *testing.T) {
	got := gnomeWindowInfo(gnomeFocusPayload{}, time.Now())
	require.Equal(t, TitleSourceNone, got.TitleSource, "expected TitleSourceNone for empty focus")
	// FinalizeWindowInfo must treat this as no foreground (not "Unknown app").
	_, ok := FinalizeWindowInfo(got)
	require.False(t, ok, "expected empty focus to be dropped by FinalizeWindowInfo")
}
