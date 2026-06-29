package platform

import "testing"

func TestFinalizeWindowInfoUsesPathFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		AppPath:     "/Applications/Safari.app",
		TitleSource: TitleSourceAX,
	})

	if !ok {
		t.Fatal("expected foreground sample")
	}
	if info.AppName != "Safari" {
		t.Fatalf("expected app name Safari, got %q", info.AppName)
	}
}

func TestFinalizeWindowInfoUsesIdentifierFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		AppIdentifier: "com.apple.Safari",
		TitleSource:   TitleSourceAX,
	})

	if !ok {
		t.Fatal("expected foreground sample")
	}
	if info.AppName != "Safari" {
		t.Fatalf("expected app name Safari, got %q", info.AppName)
	}
}

func TestFinalizeWindowInfoUsesWindowTitleFallback(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		WindowTitle: "Personal — Instagram",
		TitleSource: TitleSourceWindowAPI,
	})

	if !ok {
		t.Fatal("expected foreground sample")
	}
	if info.AppName != "Personal — Instagram" {
		t.Fatalf("expected title fallback, got %q", info.AppName)
	}
}

func TestFinalizeWindowInfoUsesUnknownForForegroundSignalWithoutNames(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{
		PID:         42,
		TitleSource: TitleSourceWindowAPI,
	})

	if !ok {
		t.Fatal("expected foreground sample")
	}
	if info.AppName != UnknownAppName {
		t.Fatalf("expected unknown app fallback, got %q", info.AppName)
	}
}

func TestFinalizeWindowInfoIgnoresEmptySample(t *testing.T) {
	info, ok := FinalizeWindowInfo(WindowInfo{})

	if ok {
		t.Fatalf("expected empty sample to be ignored, got %#v", info)
	}
}
