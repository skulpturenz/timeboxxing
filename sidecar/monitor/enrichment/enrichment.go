package enrichment

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
)

type Enricher = func(ctx context.Context, foregroundProcess monitor.ForegroundProcess) (monitor.ForegroundProcess, bool)

func Pipe(enrichers ...Enricher) Enricher {
	return func(ctx context.Context, foregroundProcess monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		acc := foregroundProcess
		some := false
		for _, fn := range enrichers {
			if enriched, ok := fn(ctx, acc); ok {
				acc = enriched
				some = true
			}
		}

		return acc, some
	}
}

func Or(enrichers ...Enricher) Enricher {
	return func(ctx context.Context, foregroundProcess monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		for _, fn := range enrichers {
			if enriched, ok := fn(ctx, foregroundProcess); ok {
				return enriched, ok
			}
		}

		return foregroundProcess, false
	}
}
