//go:build windows

package platform

import "testing"

func TestWindowsAppIdentityMapsKnownExecutable(t *testing.T) {
	identifier, appName := windowsAppIdentity(`C:\Users\me\AppData\Local\Programs\Microsoft VS Code\Code.exe`)
	if identifier != "code" {
		t.Fatalf("expected identifier code, got %q", identifier)
	}
	if appName != "Visual Studio Code" {
		t.Fatalf("expected Visual Studio Code, got %q", appName)
	}
}

func TestWindowsAppIdentityFallsBackToCapitalizedExecutable(t *testing.T) {
	identifier, appName := windowsAppIdentity(`C:\Tools\custom-helper.exe`)
	if identifier != "custom-helper" {
		t.Fatalf("expected identifier custom-helper, got %q", identifier)
	}
	if appName != "Custom-helper" {
		t.Fatalf("expected Custom-helper, got %q", appName)
	}
}

func TestWindowsAppIdentityHandlesEmptyPath(t *testing.T) {
	identifier, appName := windowsAppIdentity("")
	if identifier != "" || appName != "" {
		t.Fatalf("expected empty identity, got identifier=%q appName=%q", identifier, appName)
	}
}
