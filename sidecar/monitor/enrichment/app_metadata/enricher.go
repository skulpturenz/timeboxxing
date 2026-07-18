package appmetadata

import (
	"context"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/memo"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment"
)

// App metadata changes rarely, so memoize it for a good while rather than
// re-parsing a bundle/exe (or re-hitting a feed) on every ~200ms poll.
const (
	memoTTL     = 15 * time.Minute
	memoCleanup = 20 * time.Minute
)

// Memoized wraps an enrichment.Enricher so a given app is resolved at most once per TTL.
// It is backed by the shared memo package (TTL cache + singleflight dedup of
// concurrent lookups). Processes with no stable identity bypass the cache and
// run inner directly.
func Memoized(inner enrichment.Enricher) enrichment.Enricher {
	ttl := memoTTL
	cleanup := memoCleanup
	cache := memo.NewWithOptions(&ttl, &cleanup)

	return func(ctx context.Context, fp monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		key := identityKey(fp)
		if key == "" {
			return inner(ctx, fp)
		}

		value, _, _ := cache.Do(key, func() (any, error) {
			// Cache whatever metadata the process carries after inner ran
			// (including contributions merged by earlier enrichment.Enrichers).
			result, _ := inner(ctx, fp)
			metadata, _ := GetMetadata(result)
			return metadata, nil
		})

		metadata, ok := value.(Metadata)
		if !ok || metadata.empty() {
			return fp, false
		}
		return setMetadata(fp, metadata)
	}
}
