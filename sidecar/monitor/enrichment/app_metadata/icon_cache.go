package appmetadata

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

// identityKey picks the most stable identifier available for an app. Two samples
// of the same app share a key so caching and icon filenames stay consistent.
func identityKey(fp monitor.ForegroundProcess) string {
	for _, candidate := range []*string{fp.AppIdentifier, fp.AppPath, fp.AppName} {
		if candidate != nil {
			if value := strings.TrimSpace(*candidate); value != "" {
				return value
			}
		}
	}
	return ""
}

// iconCachePath builds a stable, filesystem-safe icon path for an app identity,
// creating the cache dir (<user cache>/timeboxxing/app-icons) if needed. The
// identity is hashed so arbitrary bundle ids / exe paths / app-ids can't produce
// an invalid or colliding filename.
func iconCachePath(identity string, ext string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "timeboxxing", "app-icons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(identity))
	name := hex.EncodeToString(sum[:]) + "." + strings.TrimPrefix(ext, ".")
	return filepath.Join(dir, name), nil
}
