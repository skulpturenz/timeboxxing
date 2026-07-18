//go:build darwin

package appmetadata

import (
	"context"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

func fpForBundle(path string) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{
		AppIdentifier: new("test.bundle." + path),
		AppPath:       new(path),
		Enrichments:   map[string]any{},
	}
}

func TestDarwinLocalMetadata_RealBundle(t *testing.T) {
	const bundle = "/System/Applications/Calculator.app"
	if _, err := os.Stat(bundle); err != nil {
		t.Skipf("bundle not present: %v", err)
	}

	out, ok := LocalMetadataEnricher(context.Background(), fpForBundle(bundle))
	require.True(t, ok, "expected enrichment for Calculator.app")
	metadata, ok := GetMetadata(out)
	require.True(t, ok, "metadata not stored in bag")

	assert.Equal(t, "Calculator", metadata.FriendlyName)
	assert.Equal(t, SourceBundle, metadata.Source)
	require.NotEmpty(t, metadata.IconPath, "expected an extracted icon path")

	// The cached icon must be a decodable PNG.
	f, err := os.Open(metadata.IconPath)
	require.NoError(t, err, "open cached icon")
	defer f.Close()
	_, err = png.Decode(f)
	require.NoError(t, err, "cached icon is not a valid PNG")

	t.Logf("Calculator => name=%q category=%q(%s) icon=%s",
		metadata.FriendlyName, metadata.Category.Label(), metadata.Category.String(), metadata.IconPath)
}

func TestBundleRoot(t *testing.T) {
	cases := map[string]string{
		"/Applications/Foo.app":                    "/Applications/Foo.app",
		"/Applications/Foo.app/Contents/MacOS/foo": "/Applications/Foo.app",
		"/usr/local/bin/somebinary":                "",
		"":                                         "",
	}
	for in, want := range cases {
		assert.Equalf(t, want, bundleRoot(in), "bundleRoot(%q)", in)
	}
}
