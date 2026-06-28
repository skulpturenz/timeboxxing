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
