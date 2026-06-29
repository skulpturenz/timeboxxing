package semantic

import (
	"context"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type IndexState string

const (
	IndexStateReady       IndexState = "ready"
	IndexStateIndexing    IndexState = "indexing"
	IndexStateEmpty       IndexState = "empty"
	IndexStateUnavailable IndexState = "unavailable"
)

type IndexStatus struct {
	State               IndexState
	CompletedEventCount int64
	IndexedEventCount   int64
	PendingEventCount   int64
	BackfillRunning     bool
	Message             string
}

type IndexStatusService struct {
	querier        queries.Querier
	backfilling    *BackfillCoordinator
	embeddingModel string
}

func NewIndexStatusService(querier queries.Querier, backfilling *BackfillCoordinator, embeddingModel string) *IndexStatusService {
	return &IndexStatusService{querier: querier, backfilling: backfilling, embeddingModel: embeddingModel}
}

func (s *IndexStatusService) Status(ctx context.Context) (IndexStatus, error) {
	if s == nil || s.querier == nil {
		return IndexStatus{
			State:   IndexStateUnavailable,
			Message: "Semantic index is unavailable.",
		}, nil
	}

	counts, err := s.querier.GetSemanticIndexCounts(ctx, s.embeddingModel)
	if err != nil {
		return IndexStatus{}, fmt.Errorf("get semantic index counts: %w", err)
	}
	running := false
	if s.backfilling != nil {
		running = s.backfilling.Status().Running
	}

	indexed := counts.IndexedEventCount
	if counts.EmbeddedEventCount < indexed {
		indexed = counts.EmbeddedEventCount
	}
	pending := counts.CompletedEventCount - indexed
	if pending < 0 {
		pending = 0
	}

	status := IndexStatus{
		CompletedEventCount: counts.CompletedEventCount,
		IndexedEventCount:   indexed,
		PendingEventCount:   pending,
		BackfillRunning:     running,
	}

	switch {
	case counts.CompletedEventCount == 0:
		status.State = IndexStateEmpty
		status.Message = "No completed usage events yet."
	case pending > 0 || running:
		status.State = IndexStateIndexing
		status.Message = "Indexing usage history..."
	default:
		status.State = IndexStateReady
		status.Message = "Semantic index is ready."
	}

	return status, nil
}
