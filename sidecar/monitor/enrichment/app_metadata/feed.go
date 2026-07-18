package appmetadata

import (
	"context"
	"os"
	"path"
	"strings"
	"time"

	"resty.dev/v3"
)

// feedTimeout bounds a single feed lookup attempt. Feeds are a best-effort
// fallback, so a slow endpoint must never stall the enrichment chain.
const feedTimeout = 4 * time.Second

// maxIconBytes caps how much of a remote icon we will download.
const maxIconBytes = 2 << 20 // 2 MiB

// feedClient is shared across feed enrichers. It retries transient failures
// (transport errors, 429, 5xx) with backoff, but never a 4xx — Flathub/Winget
// return 404 for unknown apps, which must stay a fast negative.
var feedClient = resty.New().
	SetTimeout(feedTimeout).
	SetResponseBodyLimit(maxIconBytes).
	SetRetryCount(2).
	SetRetryWaitTime(200*time.Millisecond).
	SetRetryMaxWaitTime(2*time.Second).
	SetRetryDefaultConditions(true).
	AddRetryConditions(resty.RetryConditionStatusTooManyRequests, resty.RetryConditionStatus5XX)

// getJSON fetches url and decodes the JSON body into out. Non-2xx responses are
// treated as "not found" (returns false, nil).
func getJSON(ctx context.Context, url string, out any) (bool, error) {
	resp, err := feedClient.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetResult(out).
		Get(url)
	if err != nil {
		return false, err
	}
	if !resp.IsStatusSuccess() {
		return false, nil
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

	// The client caps the body at maxIconBytes (SetResponseBodyLimit).
	resp, err := feedClient.R().SetContext(ctx).Get(url)
	if err != nil || !resp.IsStatusSuccess() {
		return ""
	}

	data := resp.Bytes()
	if len(data) == 0 {
		return ""
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return ""
	}
	return target
}
