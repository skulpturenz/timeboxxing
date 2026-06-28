//go:build darwin

package platform

import (
	"os"
	"testing"
)

func TestPathFromBundleOrProcessPrefersBundlePath(t *testing.T) {
	got := pathFromBundleOrProcess("/Applications/System Settings.app", int32(os.Getpid()))
	if got != "/Applications/System Settings.app" {
		t.Fatalf("expected bundle path to win, got %q", got)
	}
}

func TestPathFromBundleOrProcessFallsBackToExecutablePath(t *testing.T) {
	got := pathFromBundleOrProcess("", int32(os.Getpid()))
	if got == "" {
		t.Fatal("expected executable path for current process")
	}
}
