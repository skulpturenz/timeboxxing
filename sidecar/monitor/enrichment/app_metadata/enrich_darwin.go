//go:build darwin

package appmetadata

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	icns "github.com/jackmordaunt/icns/v2"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"howett.net/plist"
)

type infoPlist struct {
	DisplayName string `plist:"CFBundleDisplayName"`
	BundleName  string `plist:"CFBundleName"`
	Category    string `plist:"LSApplicationCategoryType"`
	IconFile    string `plist:"CFBundleIconFile"`
	IconName    string `plist:"CFBundleIconName"`
}

func LocalMetadataEnricher(ctx context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
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
	metadata.FriendlyName = utils.
		Coalesce(utils.Or(func(x string) bool { return !utils.IsEmptyString(x) },
			info.DisplayName,
			info.BundleName), "")
	if category, err := ParseAppleCategory(info.Category); err == nil {
		metadata.Category = category
	}
	if iconPath := extractDarwinIcon(bundle, info, identityKey(fp)); iconPath != "" {
		metadata.IconPath = iconPath
	}

	if metadata.empty() {
		return fp, false
	}

	updated := fp
	updated.Enrichments[KeyMetadata] = &metadata
	return updated, true
}

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
