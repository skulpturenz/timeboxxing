//go:build windows

package platform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsAppIdentityMapsKnownExecutable(t *testing.T) {
	identifier, appName := windowsAppIdentity(`C:\Users\me\AppData\Local\Programs\Microsoft VS Code\Code.exe`)
	require.Equal(t, "code", identifier)
	require.Equal(t, "Visual Studio Code", appName)
}

func TestWindowsAppIdentityFallsBackToCapitalizedExecutable(t *testing.T) {
	identifier, appName := windowsAppIdentity(`C:\Tools\custom-helper.exe`)
	require.Equal(t, "custom-helper", identifier)
	require.Equal(t, "Custom-helper", appName)
}

func TestWindowsAppIdentityPreservesJavaRuntimeName(t *testing.T) {
	identifier, appName := windowsAppIdentity(`C:\Program Files\Eclipse Adoptium\jdk\bin\java.exe`)
	require.Equal(t, "java", identifier)
	require.Equal(t, "java", appName)
}

func TestFinalizeWindowInfoUsesWindowsTitleWhenPathMissing(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		WindowTitle: "Untitled - Notepad",
		TitleSource: TitleSourceWindowAPI,
	})

	require.True(t, ok, "expected finalized foreground sample")
	require.Equal(t, "Untitled - Notepad", info.AppName, "expected title fallback")
}

func TestWindowsAppIdentityHandlesEmptyPath(t *testing.T) {
	identifier, appName := windowsAppIdentity("")
	require.Empty(t, identifier, "expected empty identity")
	require.Empty(t, appName, "expected empty identity")
}
