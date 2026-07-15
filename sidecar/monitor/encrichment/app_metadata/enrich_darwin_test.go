//go:build darwin

package app_metadata

import (
	"context"
	"image/png"
	"os"
	"testing"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

func strptr(s string) *string { return &s }

func fpForBundle(path string) sessionnew.ForegroundProcess {
	return sessionnew.ForegroundProcess{
		AppIdentifier: strptr("test.bundle." + path),
		AppPath:       strptr(path),
		Enrichments:   map[string]any{},
	}
}

func TestDarwinLocalMetadata_RealBundle(t *testing.T) {
	const bundle = "/System/Applications/Calculator.app"
	if _, err := os.Stat(bundle); err != nil {
		t.Skipf("bundle not present: %v", err)
	}

	out, ok := darwinLocalMetadata(context.Background(), fpForBundle(bundle))
	if !ok {
		t.Fatal("expected enrichment for Calculator.app")
	}
	metadata, ok := GetMetadata(out)
	if !ok {
		t.Fatal("metadata not stored in bag")
	}

	if metadata.FriendlyName != "Calculator" {
		t.Errorf("FriendlyName = %q, want Calculator", metadata.FriendlyName)
	}
	if metadata.Source != SourceBundle {
		t.Errorf("Source = %q, want %q", metadata.Source, SourceBundle)
	}
	if metadata.IconPath == "" {
		t.Fatal("expected an extracted icon path")
	}

	// The cached icon must be a decodable PNG.
	f, err := os.Open(metadata.IconPath)
	if err != nil {
		t.Fatalf("open cached icon: %v", err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("cached icon is not a valid PNG: %v", err)
	}

	t.Logf("Calculator => name=%q category=%q(%s) icon=%s",
		metadata.FriendlyName, metadata.CategoryLabel, metadata.CategoryCode, metadata.IconPath)
}

func TestBundleRoot(t *testing.T) {
	cases := map[string]string{
		"/Applications/Foo.app":                        "/Applications/Foo.app",
		"/Applications/Foo.app/Contents/MacOS/foo":     "/Applications/Foo.app",
		"/usr/local/bin/somebinary":                    "",
		"":                                             "",
	}
	for in, want := range cases {
		if got := bundleRoot(in); got != want {
			t.Errorf("bundleRoot(%q) = %q, want %q", in, got, want)
		}
	}
}
