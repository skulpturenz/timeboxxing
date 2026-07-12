//go:build linux

package platform

import (
	"testing"
	"time"
)

func TestParseWMClassPrefersClassName(t *testing.T) {
	got := parseWMClass([]byte("code\x00Code\x00"))
	if got != "Code" {
		t.Fatalf("expected class name Code, got %q", got)
	}
}

func TestParseWMClassFallsBackToInstanceName(t *testing.T) {
	got := parseWMClass([]byte("firefox\x00\x00"))
	if got != "firefox" {
		t.Fatalf("expected instance name firefox, got %q", got)
	}
}

func TestLinuxAppIdentifierPrefersWMClass(t *testing.T) {
	got := linuxAppIdentifier("code", "/usr/share/code/code", "Code")
	if got != "Code" {
		t.Fatalf("expected WM_CLASS identifier, got %q", got)
	}
}

func TestLinuxAppIdentifierFallsBackToExecutablePath(t *testing.T) {
	got := linuxAppIdentifier("code", "/usr/share/code/code", "")
	if got != "code" {
		t.Fatalf("expected executable basename, got %q", got)
	}
}

func TestNormalizeLinuxAppIdentityPrefersJavaRuntime(t *testing.T) {
	got := normalizeLinuxAppIdentity("java", "/usr/lib/jvm/temurin/bin/java", "jetbrains-idea", "Project")

	if got.AppName != "java" {
		t.Fatalf("expected java app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "java" {
		t.Fatalf("expected java identifier, got %q", got.AppIdentifier)
	}
	if got.AppPath != "/usr/lib/jvm/temurin/bin/java" {
		t.Fatalf("expected executable path, got %q", got.AppPath)
	}
}

func TestNormalizeLinuxAppIdentityPreservesWindowClassForNormalApps(t *testing.T) {
	got := normalizeLinuxAppIdentity("code", "/usr/share/code/code", "Code", "main.go")

	if got.AppName != "code" {
		t.Fatalf("expected app name code, got %q", got.AppName)
	}
	if got.AppIdentifier != "Code" {
		t.Fatalf("expected WM_CLASS identifier, got %q", got.AppIdentifier)
	}
	if got.AppPath != "/usr/share/code/code" {
		t.Fatalf("expected executable path, got %q", got.AppPath)
	}
}

func TestNormalizeLinuxAppIdentityFallsBackToWindowClass(t *testing.T) {
	got := normalizeLinuxAppIdentity("", "", "Code", "main.go")

	if got.AppName != "Code" {
		t.Fatalf("expected WM_CLASS app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "Code" {
		t.Fatalf("expected WM_CLASS identifier, got %q", got.AppIdentifier)
	}
}

func TestNormalizeLinuxAppIdentityFallsBackToWindowTitle(t *testing.T) {
	got := normalizeLinuxAppIdentity("", "", "", "Personal — Instagram")

	if got.AppName != "Personal — Instagram" {
		t.Fatalf("expected window title app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "Personal — Instagram" {
		t.Fatalf("expected title identifier, got %q", got.AppIdentifier)
	}
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
			if got := isWaylandSession(); got != tc.want {
				t.Fatalf("isWaylandSession() = %v, want %v", got, tc.want)
			}
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
			if got := desktopIsGNOME(); got != tc.want {
				t.Fatalf("desktopIsGNOME() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWaylandWindowInfoDerivesAppNameFromReverseDNS(t *testing.T) {
	got := waylandWindowInfo("org.mozilla.firefox", "Reddit — Mozilla Firefox")
	if got.AppName != "firefox" {
		t.Fatalf("expected app name firefox, got %q", got.AppName)
	}
	if got.AppIdentifier != "org.mozilla.firefox" {
		t.Fatalf("expected app_id identifier, got %q", got.AppIdentifier)
	}
	if got.WindowTitle != "Reddit — Mozilla Firefox" {
		t.Fatalf("expected window title preserved, got %q", got.WindowTitle)
	}
	if got.PID != 0 || got.AppPath != "" {
		t.Fatalf("expected no PID/path from wlr protocol, got pid=%d path=%q", got.PID, got.AppPath)
	}
}

func TestWaylandWindowInfoSimpleAppId(t *testing.T) {
	got := waylandWindowInfo("code", "main.go - timeboxxing")
	if got.AppName != "code" {
		t.Fatalf("expected app name code, got %q", got.AppName)
	}
	if got.AppIdentifier != "code" {
		t.Fatalf("expected identifier code, got %q", got.AppIdentifier)
	}
}

func TestWaylandWindowInfoFallsBackToTitle(t *testing.T) {
	got := waylandWindowInfo("", "Some Window")
	if got.AppName != "Some Window" {
		t.Fatalf("expected title fallback app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "" {
		t.Fatalf("expected empty identifier, got %q", got.AppIdentifier)
	}
}

func TestGnomeWindowInfoParsesPayload(t *testing.T) {
	got := gnomeWindowInfo(gnomeFocusPayload{WMClass: "Navigator", Title: "Reddit - Firefox", PID: 0}, time.Now())
	if got.AppName != "Navigator" {
		t.Fatalf("expected app name from wm_class, got %q", got.AppName)
	}
	if got.AppIdentifier != "Navigator" {
		t.Fatalf("expected identifier from wm_class, got %q", got.AppIdentifier)
	}
	if got.WindowTitle != "Reddit - Firefox" {
		t.Fatalf("expected window title, got %q", got.WindowTitle)
	}
}

func TestGnomeWindowInfoEmptyFocusReportsNone(t *testing.T) {
	got := gnomeWindowInfo(gnomeFocusPayload{}, time.Now())
	if got.TitleSource != TitleSourceNone {
		t.Fatalf("expected TitleSourceNone for empty focus, got %q", got.TitleSource)
	}
	// FinalizeWindowInfo must treat this as no foreground (not "Unknown app").
	if _, ok := FinalizeWindowInfo(got); ok {
		t.Fatalf("expected empty focus to be dropped by FinalizeWindowInfo")
	}
}
