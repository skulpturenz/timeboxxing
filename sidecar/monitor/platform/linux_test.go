//go:build linux

package platform

import "testing"

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
