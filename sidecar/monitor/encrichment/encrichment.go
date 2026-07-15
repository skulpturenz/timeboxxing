package encrichment

import (
	"context"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
)

type Enricher = func(ctx context.Context, foregroundProcess sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool)

func Pipe(enrichers ...Enricher) Enricher {
	return func(ctx context.Context, foregroundProcess sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
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
