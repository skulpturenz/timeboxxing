package sqlitevectorupdate

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const DefaultReleaseURL = "https://api.github.com/repos/sqliteai/sqlite-vector/releases/latest"

type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type Target struct {
	DisplayName string
	AssetPrefix string
	OutputPath  string
	ZipMember   string
}

type SelectedAsset struct {
	Target Target
	Asset  Asset
}

type UpdatedFile struct {
	Target Target
	SHA256 string
}

type Updater struct {
	Client     *http.Client
	ReleaseURL string
}

var Targets = []Target{
	{
		DisplayName: "macOS arm64",
		AssetPrefix: "vector-macos-arm64",
		OutputPath:  filepath.Join("darwin-arm64", "vector.dylib"),
		ZipMember:   "vector.dylib",
	},
	{
		DisplayName: "macOS amd64",
		AssetPrefix: "vector-macos-x86_64",
		OutputPath:  filepath.Join("darwin-amd64", "vector.dylib"),
		ZipMember:   "vector.dylib",
	},
	{
		DisplayName: "Linux arm64",
		AssetPrefix: "vector-linux-arm64",
		OutputPath:  filepath.Join("linux-arm64", "vector.so"),
		ZipMember:   "vector.so",
	},
	{
		DisplayName: "Linux amd64",
		AssetPrefix: "vector-linux-x86_64",
		OutputPath:  filepath.Join("linux-amd64", "vector.so"),
		ZipMember:   "vector.so",
	},
	{
		DisplayName: "Windows amd64",
		AssetPrefix: "vector-windows-x86_64",
		OutputPath:  filepath.Join("windows-amd64", "vector.dll"),
		ZipMember:   "vector.dll",
	},
}

func (u Updater) Update(ctx context.Context, outputDir string) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("output directory is required")
	}

	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}
	releaseURL := strings.TrimSpace(u.ReleaseURL)
	if releaseURL == "" {
		releaseURL = DefaultReleaseURL
	}

	release, err := FetchRelease(ctx, client, releaseURL)
	if err != nil {
		return err
	}
	selections, err := SelectAssets(release, Targets)
	if err != nil {
		return err
	}

	files := make([]struct {
		target Target
		data   []byte
		hash   string
	}, 0, len(selections))
	for _, selection := range selections {
		archiveData, err := DownloadAsset(ctx, client, selection.Asset)
		if err != nil {
			return err
		}
		if err := VerifyDigest(archiveData, selection.Asset.Digest); err != nil {
			return fmt.Errorf("verify %s: %w", selection.Asset.Name, err)
		}
		extensionData, err := ExtractZipMember(archiveData, selection.Target.ZipMember)
		if err != nil {
			return fmt.Errorf("extract %s from %s: %w", selection.Target.ZipMember, selection.Asset.Name, err)
		}
		files = append(files, struct {
			target Target
			data   []byte
			hash   string
		}{
			target: selection.Target,
			data:   extensionData,
			hash:   SHA256Hex(extensionData),
		})
	}

	updated := make([]UpdatedFile, 0, len(files))
	for _, file := range files {
		if err := WriteFileAtomic(filepath.Join(outputDir, file.target.OutputPath), file.data, 0o644); err != nil {
			return err
		}
		updated = append(updated, UpdatedFile{Target: file.target, SHA256: file.hash})
	}

	readme := RenderREADME(release, updated)
	if err := WriteFileAtomic(filepath.Join(outputDir, "README.md"), []byte(readme), 0o644); err != nil {
		return err
	}
	return nil
}

func FetchRelease(ctx context.Context, client *http.Client, releaseURL string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "timeboxxing-sqlite-vector-updater")

	response, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("fetch sqlite-vector release: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Release{}, fmt.Errorf("fetch sqlite-vector release: unexpected HTTP status %s", response.Status)
	}

	var release Release
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("decode sqlite-vector release: %w", err)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return Release{}, fmt.Errorf("sqlite-vector release has no tag name")
	}
	return release, nil
}

func SelectAssets(release Release, targets []Target) ([]SelectedAsset, error) {
	selections := make([]SelectedAsset, 0, len(targets))
	for _, target := range targets {
		var matches []Asset
		for _, asset := range release.Assets {
			if strings.HasPrefix(asset.Name, target.AssetPrefix+"-") && strings.HasSuffix(asset.Name, ".zip") {
				matches = append(matches, asset)
			}
		}
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("sqlite-vector release %q is missing asset %s-*.zip", release.TagName, target.AssetPrefix)
		case 1:
			selections = append(selections, SelectedAsset{Target: target, Asset: matches[0]})
		default:
			return nil, fmt.Errorf("sqlite-vector release %q has multiple assets matching %s-*.zip", release.TagName, target.AssetPrefix)
		}
	}
	return selections, nil
}

func DownloadAsset(ctx context.Context, client *http.Client, asset Asset) ([]byte, error) {
	if strings.TrimSpace(asset.BrowserDownloadURL) == "" {
		return nil, fmt.Errorf("asset %q has no download URL", asset.Name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "timeboxxing-sqlite-vector-updater")

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", asset.Name, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: unexpected HTTP status %s", asset.Name, response.Status)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", asset.Name, err)
	}
	return data, nil
}

func VerifyDigest(data []byte, digest string) error {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return nil
	}

	algorithm, value, ok := strings.Cut(digest, ":")
	if !ok {
		return fmt.Errorf("unsupported digest format %q", digest)
	}
	if !strings.EqualFold(algorithm, "sha256") {
		return fmt.Errorf("unsupported digest algorithm %q", algorithm)
	}

	expected, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("decode sha256 digest: %w", err)
	}
	actual := sha256.Sum256(data)
	if !bytes.Equal(actual[:], expected) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", hex.EncodeToString(actual[:]), strings.ToLower(value))
	}
	return nil
}

func ExtractZipMember(data []byte, memberName string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	for _, file := range reader.File {
		if file.Name != memberName {
			continue
		}
		if file.FileInfo().IsDir() {
			return nil, fmt.Errorf("%s is a directory", memberName)
		}
		return readZipFile(file)
	}
	return nil, fmt.Errorf("zip member %q not found", memberName)
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory for %s: %w", path, err)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary file for %s: %w", path, err)
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temporary file for %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	cleanup = false
	return nil
}

func RenderREADME(release Release, files []UpdatedFile) string {
	var builder strings.Builder
	tagName := strings.TrimSpace(release.TagName)
	releaseURL := strings.TrimSpace(release.HTMLURL)

	builder.WriteString("# SQLite-Vector Native Extensions\n\n")
	builder.WriteString("These native extension binaries are from\n")
	builder.WriteString("`sqliteai/sqlite-vector` release `")
	builder.WriteString(tagName)
	builder.WriteString("`:\n\n")
	builder.WriteString(releaseURL)
	builder.WriteString("\n\n")
	builder.WriteString("Bundled files:\n\n")
	builder.WriteString("| Platform | File | SHA-256 |\n")
	builder.WriteString("| --- | --- | --- |\n")
	for _, file := range files {
		builder.WriteString("| ")
		builder.WriteString(file.Target.DisplayName)
		builder.WriteString(" | `")
		builder.WriteString(filepath.ToSlash(file.Target.OutputPath))
		builder.WriteString("` | `")
		builder.WriteString(file.SHA256)
		builder.WriteString("` |\n")
	}
	builder.WriteString("\n")
	builder.WriteString("Before production distribution, confirm the upstream license obligations for\n")
	builder.WriteString("this project.\n")
	return builder.String()
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
