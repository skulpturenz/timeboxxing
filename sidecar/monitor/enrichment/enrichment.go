package enrichment

import (
	"context"
	"maps"

	"dario.cat/mergo"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
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

func Merge(enrichers ...Enricher) Enricher {
	return func(ctx context.Context, foregroundProcess monitor.ForegroundProcess) (monitor.ForegroundProcess, bool) {
		clone := func(fp monitor.ForegroundProcess) monitor.ForegroundProcess {
			dup := fp
			dup.Enrichments = maps.Clone(fp.Enrichments)
			if dup.Enrichments == nil {
				dup.Enrichments = map[string]any{}
			}
			return dup
		}

		acc := foregroundProcess
		enrich := utils.ParallelMapWithClone(clone, enrichers...)

		enrichments, ok := enrich(ctx, acc)
		if !ok {
			return foregroundProcess, false
		}

		for _, v := range enrichments {
			if err := mergo.Merge(&acc.Enrichments, v.Enrichments); err != nil {
				return foregroundProcess, false
			}
		}

		return acc, true
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
