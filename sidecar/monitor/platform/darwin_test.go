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

func TestNormalizeDarwinAppIdentityPrefersJavaExecutable(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"idea",
		"com.jetbrains.intellij",
		"/Applications/IntelliJ IDEA.app",
		"/Library/Java/JavaVirtualMachines/temurin.jdk/Contents/Home/bin/java",
	)

	if got.AppName != "java" {
		t.Fatalf("expected java app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "java" {
		t.Fatalf("expected java identifier, got %q", got.AppIdentifier)
	}
	if got.AppPath != "/Library/Java/JavaVirtualMachines/temurin.jdk/Contents/Home/bin/java" {
		t.Fatalf("expected executable path, got %q", got.AppPath)
	}
}

func TestNormalizeDarwinAppIdentityFallsBackFromBlankName(t *testing.T) {
	identity := normalizeDarwinAppIdentity(
		"",
		"com.apple.Safari",
		"/Applications/Safari.app",
		"/Applications/Safari.app/Contents/MacOS/Safari",
	)
	info, ok := FinalizeWindowInfo(WindowInfo{
		AppName:       identity.AppName,
		AppIdentifier: identity.AppIdentifier,
		AppPath:       identity.AppPath,
		TitleSource:   TitleSourceAX,
	})

	if !ok {
		t.Fatal("expected finalized foreground sample")
	}
	if info.AppName != "Safari" {
		t.Fatalf("expected Safari fallback, got %q", info.AppName)
	}
}

func TestNormalizeDarwinAppIdentityPreservesBundleApps(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"Visual Studio Code",
		"com.microsoft.VSCode",
		"/Applications/Visual Studio Code.app",
		"/Applications/Visual Studio Code.app/Contents/MacOS/Code",
	)

	if got.AppName != "Visual Studio Code" {
		t.Fatalf("expected bundle app name, got %q", got.AppName)
	}
	if got.AppIdentifier != "com.microsoft.VSCode" {
		t.Fatalf("expected bundle identifier, got %q", got.AppIdentifier)
	}
	if got.AppPath != "/Applications/Visual Studio Code.app" {
		t.Fatalf("expected bundle path, got %q", got.AppPath)
	}
}

func TestNormalizeDarwinAppIdentityFallsBackToExecutablePath(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"helper",
		"",
		"",
		"/usr/local/bin/helper",
	)

	if got.AppName != "helper" {
		t.Fatalf("expected app name to be preserved, got %q", got.AppName)
	}
	if got.AppPath != "/usr/local/bin/helper" {
		t.Fatalf("expected executable path fallback, got %q", got.AppPath)
	}
}
