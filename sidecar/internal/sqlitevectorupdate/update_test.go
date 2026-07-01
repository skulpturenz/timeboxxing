package sqlitevectorupdate

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSelectAssets(t *testing.T) {
	release := Release{
		TagName: "1.2.3",
		Assets: []Asset{
			{Name: "vector-macos-1.2.3.zip"},
			{Name: "vector-macos-arm64-1.2.3.tar.gz"},
			{Name: "vector-macos-arm64-1.2.3.zip", BrowserDownloadURL: "https://example.test/darwin-arm64"},
			{Name: "vector-macos-x86_64-1.2.3.zip", BrowserDownloadURL: "https://example.test/darwin-amd64"},
			{Name: "vector-linux-arm64-1.2.3.zip", BrowserDownloadURL: "https://example.test/linux-arm64"},
			{Name: "vector-linux-x86_64-1.2.3.zip", BrowserDownloadURL: "https://example.test/linux-amd64"},
			{Name: "vector-windows-x86_64-1.2.3.zip", BrowserDownloadURL: "https://example.test/windows-amd64"},
		},
	}

	selections, err := SelectAssets(release, Targets)
	if err != nil {
		t.Fatalf("select assets: %v", err)
	}
	if len(selections) != len(Targets) {
		t.Fatalf("expected %d selections, got %d", len(Targets), len(selections))
	}
	for index, selection := range selections {
		if selection.Target != Targets[index] {
			t.Fatalf("selection %d target mismatch: got %+v want %+v", index, selection.Target, Targets[index])
		}
		if !strings.HasPrefix(selection.Asset.Name, selection.Target.AssetPrefix+"-") {
			t.Fatalf("selection %d asset %q does not match target prefix %q", index, selection.Asset.Name, selection.Target.AssetPrefix)
		}
		if !strings.HasSuffix(selection.Asset.Name, ".zip") {
			t.Fatalf("selection %d asset %q is not a zip", index, selection.Asset.Name)
		}
	}
}

func TestSelectAssetsReportsMissingAndDuplicateAssets(t *testing.T) {
	_, err := SelectAssets(Release{TagName: "1.2.3"}, []Target{Targets[0]})
	if err == nil || !strings.Contains(err.Error(), "missing asset") {
		t.Fatalf("expected missing asset error, got %v", err)
	}

	_, err = SelectAssets(Release{
		TagName: "1.2.3",
		Assets: []Asset{
			{Name: "vector-macos-arm64-1.2.3.zip"},
			{Name: "vector-macos-arm64-1.2.3-retry.zip"},
		},
	}, []Target{Targets[0]})
	if err == nil || !strings.Contains(err.Error(), "multiple assets") {
		t.Fatalf("expected duplicate asset error, got %v", err)
	}
}

func TestVerifyDigest(t *testing.T) {
	data := []byte("archive bytes")
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])

	if err := VerifyDigest(data, digest); err != nil {
		t.Fatalf("verify matching digest: %v", err)
	}
	if err := VerifyDigest(data, ""); err != nil {
		t.Fatalf("empty digest should be accepted: %v", err)
	}
	if err := VerifyDigest([]byte("different bytes"), digest); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	if err := VerifyDigest(data, "sha512:abc123"); err == nil || !strings.Contains(err.Error(), "unsupported digest algorithm") {
		t.Fatalf("expected unsupported algorithm error, got %v", err)
	}
}

func TestExtractZipMember(t *testing.T) {
	archiveData := testZip(t, map[string][]byte{
		"vector.dll": []byte("runtime extension"),
		"vector.lib": []byte("import library"),
	})

	data, err := ExtractZipMember(archiveData, "vector.dll")
	if err != nil {
		t.Fatalf("extract vector.dll: %v", err)
	}
	if string(data) != "runtime extension" {
		t.Fatalf("unexpected member data %q", string(data))
	}

	if _, err := ExtractZipMember(archiveData, "vector.so"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing member error, got %v", err)
	}
}

func TestRenderREADME(t *testing.T) {
	readme := RenderREADME(Release{
		TagName: "1.2.3",
		HTMLURL: "https://github.com/sqliteai/sqlite-vector/releases/tag/1.2.3",
	}, []UpdatedFile{
		{Target: Targets[0], SHA256: "darwin-arm64-hash"},
		{Target: Targets[4], SHA256: "windows-amd64-hash"},
	})

	for _, want := range []string{
		"`sqliteai/sqlite-vector` release `1.2.3`",
		"https://github.com/sqliteai/sqlite-vector/releases/tag/1.2.3",
		"| macOS arm64 | `darwin-arm64/vector.dylib` | `darwin-arm64-hash` |",
		"| Windows amd64 | `windows-amd64/vector.dll` | `windows-amd64-hash` |",
		"Before production distribution",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q:\n%s", want, readme)
		}
	}
}

func testZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, data := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip member %s: %v", name, err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("write zip member %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}
