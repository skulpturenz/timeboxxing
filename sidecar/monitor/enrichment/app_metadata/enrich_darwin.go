//go:build darwin

package appmetadata

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	icns "github.com/jackmordaunt/icns/v2"
	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
	"howett.net/plist"
)

// LocalMetadata reads friendly name, category, and icon for the foreground
// app out of its macOS .app bundle's Info.plist. There is no description field
// in Info.plist, so Description is left for a feed fallback. Wrap with Memoized
// to avoid re-parsing the bundle on every poll.
var LocalMetadata Enricher = darwinLocalMetadata

// infoPlist captures the Info.plist keys we care about. Missing keys decode to
// their zero value.
type infoPlist struct {
	DisplayName string `plist:"CFBundleDisplayName"`
	BundleName  string `plist:"CFBundleName"`
	Category    string `plist:"LSApplicationCategoryType"`
	IconFile    string `plist:"CFBundleIconFile"`
	IconName    string `plist:"CFBundleIconName"`
}

func darwinLocalMetadata(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
	if fp.AppPath == nil {
		return fp, false
	}
	bundle := bundleRoot(strings.TrimSpace(*fp.AppPath))
	if bundle == "" {
		return fp, false
	}

	info, err := readInfoPlist(filepath.Join(bundle, "Contents", "Info.plist"))
	if err != nil {
		return fp, false
	}

	metadata := Metadata{Source: SourceBundle}
	metadata.FriendlyName = firstNonBlank(info.DisplayName, info.BundleName)
	if code, label := CategoryFromApple(info.Category); code != "" {
		metadata.CategoryCode = code
		metadata.CategoryLabel = label
	}
	if iconPath := extractDarwinIcon(bundle, info, identityKey(fp)); iconPath != "" {
		metadata.IconPath = iconPath
	}

	if metadata.empty() {
		return fp, false
	}
	return setMetadata(fp, metadata)
}

// bundleRoot returns the ".app" bundle directory for a path that may point at
// the bundle itself or at an executable nested inside it. Returns "" when the
// path is not part of an .app bundle.
func bundleRoot(appPath string) string {
	if appPath == "" {
		return ""
	}
	if strings.HasSuffix(appPath, ".app") {
		return appPath
	}
	// Walk up looking for the ".app" component (e.g. /Applications/Foo.app/Contents/MacOS/foo).
	if idx := strings.Index(appPath, ".app/"); idx >= 0 {
		return appPath[:idx+len(".app")]
	}
	return ""
}

func readInfoPlist(path string) (infoPlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return infoPlist{}, err
	}
	var info infoPlist
	if _, err := plist.Unmarshal(data, &info); err != nil {
		return infoPlist{}, err
	}
	return info, nil
}

// extractDarwinIcon locates the bundle's .icns, decodes it, and writes a PNG to
// the icon cache. Returns the cached PNG path, or "" if no icon could be read.
func extractDarwinIcon(bundle string, info infoPlist, identity string) string {
	source := resolveICNS(filepath.Join(bundle, "Contents", "Resources"), info)
	if source == "" {
		return "" // nothing to decode
	}

	target, err := iconCachePath(identity, "png")
	if err != nil {
		return ""
	}
	if _, err := os.Stat(target); err == nil {
		return target // already cached
	}

	in, err := os.Open(source)
	if err != nil {
		return ""
	}
	defer in.Close()

	img, err := icns.Decode(in)
	if err != nil {
		return ""
	}

	out, err := os.Create(target)
	if err != nil {
		return ""
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		os.Remove(target)
		return ""
	}
	return target
}

// resolveICNS finds the .icns file inside a bundle's Resources dir, trying the
// declared icon keys first and falling back to the conventional AppIcon.icns or
// any .icns present.
func resolveICNS(resources string, info infoPlist) string {
	candidates := []string{
		info.IconFile,
		info.IconFile + ".icns",
		info.IconName + ".icns",
		"AppIcon.icns",
	}
	for _, name := range candidates {
		name = strings.TrimSpace(name)
		if name == "" || name == ".icns" {
			continue
		}
		path := filepath.Join(resources, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Last resort: first .icns in the Resources dir.
	if matches, _ := filepath.Glob(filepath.Join(resources, "*.icns")); len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
