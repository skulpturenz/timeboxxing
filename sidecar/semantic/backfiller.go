package semantic

import (
	"context"
	"fmt"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
)

const semanticBackfillDelay = 100 * time.Millisecond

type MissingTransitionEventLister interface {
	ListMissingSemanticEventDocumentIDs(ctx context.Context, arg readqueries.ListMissingSemanticEventDocumentIDsParams) ([]int64, error)
}

type TransitionEventEnqueuer interface {
	EnqueueTransitionEvent(ctx context.Context, transitionEventID int64) error
}

type Backfiller struct {
	lister         MissingTransitionEventLister
	enqueuer       TransitionEventEnqueuer
	embeddingModel string
}

type BackfillResult struct {
	Checked  int
	Enqueued int
	Failed   int
}

func NewBackfiller(lister MissingTransitionEventLister, enqueuer TransitionEventEnqueuer, embeddingModel string) *Backfiller {
	return &Backfiller{lister: lister, enqueuer: enqueuer, embeddingModel: embeddingModel}
}

func (b *Backfiller) HasMissing(ctx context.Context) (bool, error) {
	if b.lister == nil {
		return false, fmt.Errorf("missing transition event lister is required")
	}

	ids, err := b.lister.ListMissingSemanticEventDocumentIDs(ctx, readqueries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: b.embeddingModel,
		Limit:          1,
	})
	if err != nil {
		return false, fmt.Errorf("list missing semantic event documents: %w", err)
	}
	return len(ids) > 0, nil
}

func (b *Backfiller) BackfillMissing(ctx context.Context, limit int64) (BackfillResult, error) {
	if limit <= 0 {
		return BackfillResult{}, fmt.Errorf("limit must be positive")
	}
	if b.lister == nil {
		return BackfillResult{}, fmt.Errorf("missing transition event lister is required")
	}
	if b.enqueuer == nil {
		return BackfillResult{}, fmt.Errorf("transition event enqueuer is required")
	}

	ids, err := b.lister.ListMissingSemanticEventDocumentIDs(ctx, readqueries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: b.embeddingModel,
		Limit:          limit,
	})
	if err != nil {
		return BackfillResult{}, fmt.Errorf("list missing semantic event documents: %w", err)
	}

	result := BackfillResult{Checked: len(ids)}
	var firstErr error
	for index, id := range ids {
		if err := b.enqueuer.EnqueueTransitionEvent(ctx, id); err != nil {
			result.Failed++
			if firstErr == nil {
				firstErr = err
			}
		} else {
			result.Enqueued++
		}

		if index < len(ids)-1 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(semanticBackfillDelay):
			}
		}
	}
	if result.Failed > 0 {
		return result, fmt.Errorf("semantic backfill enqueue failed for %d of %d checked events: %w", result.Failed, result.Checked, firstErr)
	}

	return result, nil
}
