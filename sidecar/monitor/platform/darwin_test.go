//go:build darwin

package platform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathFromBundleOrProcessPrefersBundlePath(t *testing.T) {
	got := pathFromBundleOrProcess("/Applications/System Settings.app", int32(os.Getpid()))
	require.Equal(t, "/Applications/System Settings.app", got, "expected bundle path to win")
}

func TestPathFromBundleOrProcessFallsBackToExecutablePath(t *testing.T) {
	got := pathFromBundleOrProcess("", int32(os.Getpid()))
	require.NotEmpty(t, got, "expected executable path for current process")
}

func TestNormalizeDarwinAppIdentityPrefersJavaExecutable(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"idea",
		"com.jetbrains.intellij",
		"/Applications/IntelliJ IDEA.app",
		"/Library/Java/JavaVirtualMachines/temurin.jdk/Contents/Home/bin/java",
	)

	require.Equal(t, "java", got.AppName)
	require.Equal(t, "java", got.AppIdentifier)
	require.Equal(t, "/Library/Java/JavaVirtualMachines/temurin.jdk/Contents/Home/bin/java", got.AppPath)
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

	require.True(t, ok, "expected finalized foreground sample")
	require.Equal(t, "Safari", info.AppName, "expected Safari fallback")
}

func TestNormalizeDarwinAppIdentityPreservesBundleApps(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"Visual Studio Code",
		"com.microsoft.VSCode",
		"/Applications/Visual Studio Code.app",
		"/Applications/Visual Studio Code.app/Contents/MacOS/Code",
	)

	require.Equal(t, "Visual Studio Code", got.AppName)
	require.Equal(t, "com.microsoft.VSCode", got.AppIdentifier)
	require.Equal(t, "/Applications/Visual Studio Code.app", got.AppPath)
}

func TestNormalizeDarwinAppIdentityFallsBackToExecutablePath(t *testing.T) {
	got := normalizeDarwinAppIdentity(
		"helper",
		"",
		"",
		"/usr/local/bin/helper",
	)

	require.Equal(t, "helper", got.AppName, "expected app name to be preserved")
	require.Equal(t, "/usr/local/bin/helper", got.AppPath, "expected executable path fallback")
}
