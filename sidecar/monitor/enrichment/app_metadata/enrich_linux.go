//go:build linux

package appmetadata

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
	"gopkg.in/ini.v1"
)

func LocalMetadataEnricher(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
	identifier := ""
	if fp.AppIdentifier != nil {
		identifier = strings.TrimSpace(*fp.AppIdentifier)
	}
	exe := ""
	if fp.AppPath != nil {
		exe = strings.TrimSpace(*fp.AppPath)
	}

	entry, ok := findDesktopEntry(identifier, exe)
	if !ok {
		return fp, false
	}

	metadata := Metadata{
		Source:       SourceDesktop,
		FriendlyName: strings.TrimSpace(entry.Name),
		Description:  strings.TrimSpace(entry.Comment),
	}
	if category, err := ParseFreedesktopCategories(entry.Categories); err == nil {
		metadata.Category = category
	}
	if iconPath := cacheLinuxIcon(entry.Icon, identityKey(fp)); iconPath != "" {
		metadata.IconPath = iconPath
	}

	if metadata.empty() {
		return fp, false
	}
	return setMetadata(fp, metadata)
}

// xdgApplicationDirs returns the directories that hold .desktop files, honoring
// XDG_DATA_HOME / XDG_DATA_DIRS and including flatpak export dirs.
func xdgApplicationDirs() []string {
	var roots []string

	if home := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); home != "" {
		roots = append(roots, home)
	} else if h, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(h, ".local", "share"))
	}

	dataDirs := strings.TrimSpace(os.Getenv("XDG_DATA_DIRS"))
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	roots = append(roots, strings.Split(dataDirs, ":")...)

	// Flatpak exports (system + user) are not always in XDG_DATA_DIRS.
	if h, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(h, ".local", "share", "flatpak", "exports", "share"))
	}
	roots = append(roots, "/var/lib/flatpak/exports/share")

	dirs := make([]string, 0, len(roots))
	seen := map[string]bool{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "applications")
		if seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs
}

// findDesktopEntry locates the best-matching .desktop entry. It first tries a
// direct <identifier>.desktop hit (fast path for Wayland app_ids like
// org.videolan.VLC), then scans for an entry whose StartupWMClass, file
// basename, or Exec basename matches the identifier or executable.
func findDesktopEntry(identifier string, exe string) (desktopEntry, bool) {
	dirs := xdgApplicationDirs()
	exeBase := ""
	if exe != "" {
		exeBase = filepath.Base(exe)
	}
	idLower := strings.ToLower(identifier)

	// Fast path: <identifier>.desktop.
	if identifier != "" {
		for _, dir := range dirs {
			candidate := filepath.Join(dir, identifier+".desktop")
			cfg, err := ini.LoadSources(
				ini.LoadOptions{IgnoreInlineComment: true, SkipUnrecognizableLines: true},
				candidate,
			)
			if err != nil {
				continue
			}
			var entry desktopEntry
			if err := cfg.Section("Desktop Entry").MapTo(&entry); err != nil {
				continue
			}
			return entry, true
		}
	}

	// Scan for a match on StartupWMClass / basename / Exec.
	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".desktop") {
				continue
			}
			cfg, err := ini.LoadSources(
				ini.LoadOptions{IgnoreInlineComment: true, SkipUnrecognizableLines: true},
				filepath.Join(dir, file.Name()),
			)
			if err != nil {
				continue
			}
			var entry desktopEntry
			if err := cfg.Section("Desktop Entry").MapTo(&entry); err != nil {
				continue
			}
			base := strings.TrimSuffix(file.Name(), ".desktop")
			if desktopMatches(entry, base, idLower, exeBase) {
				return entry, true
			}
		}
	}
	return desktopEntry{}, false
}

// iconThemeDirs are the roots searched for a themed icon name.
func iconThemeDirs() []string {
	var dirs []string
	for _, appDir := range xdgApplicationDirs() {
		share := filepath.Dir(appDir) // .../share/applications -> .../share
		dirs = append(dirs,
			filepath.Join(share, "icons", "hicolor"),
			filepath.Join(share, "icons"),
			filepath.Join(share, "pixmaps"),
		)
	}
	return dirs
}

// resolveIconFile turns an Icon= value into an absolute file path. Absolute
// paths are used as-is; bare names are searched across theme dirs, preferring
// larger raster sizes then SVG then pixmaps.
func resolveIconFile(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return ""
	}
	if filepath.IsAbs(icon) {
		if _, err := os.Stat(icon); err == nil {
			return icon
		}
		return ""
	}

	// Preferred raster sizes (largest first), then scalable, then legacy pixmaps.
	sizes := []string{"512x512", "256x256", "128x128", "96x96", "64x64", "48x48", "32x32"}
	exts := []string{".png", ".svg", ".xpm"}

	for _, root := range iconThemeDirs() {
		for _, size := range sizes {
			for _, ext := range []string{".png"} {
				candidate := filepath.Join(root, size, "apps", icon+ext)
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
		// scalable/svg and flat layouts
		for _, ext := range exts {
			for _, candidate := range []string{
				filepath.Join(root, "scalable", "apps", icon+ext),
				filepath.Join(root, icon+ext),
			} {
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
	}
	return ""
}

// cacheLinuxIcon resolves the icon and copies it into the icon cache, returning
// the cached path (or "" if it could not be resolved).
func cacheLinuxIcon(icon string, identity string) string {
	source := resolveIconFile(icon)
	if source == "" {
		return ""
	}
	target, err := iconCachePath(identity, strings.TrimPrefix(filepath.Ext(source), "."))
	if err != nil {
		return ""
	}
	if _, err := os.Stat(target); err == nil {
		return target
	}
	if err := copyFile(source, target); err != nil {
		return ""
	}
	return target
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
