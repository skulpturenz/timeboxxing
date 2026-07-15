package app_metadata

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// feedTimeout bounds a single feed lookup. Feeds are a best-effort fallback, so
// a slow endpoint must never stall the enrichment chain.
const feedTimeout = 4 * time.Second

// maxIconBytes caps how much of a remote icon we will download.
const maxIconBytes = 2 << 20 // 2 MiB

// feedHTTPClient is shared across feed enrichers.
var feedHTTPClient = &http.Client{Timeout: feedTimeout}

// getJSON fetches url and decodes the JSON body into out. Non-200 responses are
// treated as "not found" (returns false, nil).
func getJSON(ctx context.Context, url string, out any) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, feedTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := feedHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return false, err
	}
	return true, nil
}

// downloadIconToCache fetches a remote icon URL into the icon cache, returning
// the cached path. Best-effort: any failure returns "".
func downloadIconToCache(ctx context.Context, url string, identity string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}

	ext := strings.TrimPrefix(strings.ToLower(path.Ext(path.Base(url))), ".")
	switch ext {
	case "png", "svg", "jpg", "jpeg", "webp", "ico":
	default:
		ext = "png"
	}

	target, err := iconCachePath(identity, ext)
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(ctx, feedTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := feedHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIconBytes))
	if err != nil || len(data) == 0 {
		return ""
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return ""
	}
	return target
}
