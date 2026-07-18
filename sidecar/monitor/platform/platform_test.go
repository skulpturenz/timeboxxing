package platform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizeWindowInfoUsesPathFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		AppPath:     "/Applications/Safari.app",
		TitleSource: TitleSourceAX,
	})

	require.True(t, ok, "expected foreground sample")
	require.Equal(t, "Safari", info.AppName)
}

func TestFinalizeWindowInfoUsesIdentifierFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		AppIdentifier: "com.apple.Safari",
		TitleSource:   TitleSourceAX,
	})

	require.True(t, ok, "expected foreground sample")
	require.Equal(t, "Safari", info.AppName)
}

func TestFinalizeWindowInfoUsesWindowTitleFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		WindowTitle: "Personal — Instagram",
		TitleSource: TitleSourceWindowAPI,
	})

	require.True(t, ok, "expected foreground sample")
	require.Equal(t, "Personal — Instagram", info.AppName, "expected title fallback")
}

func TestFinalizeWindowInfoUsesUnknownForForegroundSignalWithoutNames(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		PID:         42,
		TitleSource: TitleSourceWindowAPI,
	})

	require.True(t, ok, "expected foreground sample")
	require.Equal(t, UnknownAppName, info.AppName, "expected unknown app fallback")
}

func TestFinalizeWindowInfoIgnoresEmptySample(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{})

	require.Falsef(t, ok, "expected empty sample to be ignored, got %#v", info)
}
