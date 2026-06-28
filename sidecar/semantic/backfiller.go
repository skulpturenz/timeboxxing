package semantic

import (
	"context"
	"fmt"
	"time"
)

const semanticBackfillDelay = 100 * time.Millisecond

type MissingTransitionEventLister interface {
	ListMissingSemanticEventDocumentIDs(ctx context.Context, limit int64) ([]int64, error)
}

type TransitionEventIndexer interface {
	IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error)
}

type Backfiller struct {
	lister  MissingTransitionEventLister
	indexer TransitionEventIndexer
}

type BackfillResult struct {
	Checked int
	Indexed int
	Failed  int
}

func NewBackfiller(lister MissingTransitionEventLister, indexer TransitionEventIndexer) *Backfiller {
	return &Backfiller{lister: lister, indexer: indexer}
}

func (b *Backfiller) HasMissing(ctx context.Context) (bool, error) {
	if b.lister == nil {
		return false, fmt.Errorf("missing transition event lister is required")
	}

	ids, err := b.lister.ListMissingSemanticEventDocumentIDs(ctx, 1)
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
	if b.indexer == nil {
		return BackfillResult{}, fmt.Errorf("transition event indexer is required")
	}

	ids, err := b.lister.ListMissingSemanticEventDocumentIDs(ctx, limit)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("list missing semantic event documents: %w", err)
	}

	result := BackfillResult{Checked: len(ids)}
	for index, id := range ids {
		if _, err := b.indexer.IndexTransitionEvent(ctx, id); err != nil {
			result.Failed++
		} else {
			result.Indexed++
		}

		if index < len(ids)-1 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(semanticBackfillDelay):
			}
		}
	}

	return result, nil
}
