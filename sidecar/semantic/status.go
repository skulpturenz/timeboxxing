package semantic

import (
	"context"
	"database/sql"
	"fmt"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
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
	querier          readqueries.Querier
	backfilling      *BackfillCoordinator
	embeddingModelID int64
}

func NewIndexStatusService(querier readqueries.Querier, backfilling *BackfillCoordinator, embeddingModelID int64) *IndexStatusService {
	return &IndexStatusService{querier: querier, backfilling: backfilling, embeddingModelID: embeddingModelID}
}

func (s *IndexStatusService) Status(ctx context.Context) (IndexStatus, error) {
	if s == nil || s.querier == nil {
		return IndexStatus{
			State:   IndexStateUnavailable,
			Message: "Semantic index is unavailable.",
		}, nil
	}

	counts, err := s.querier.GetSemanticIndexCounts(ctx, sql.NullInt64{Int64: s.embeddingModelID, Valid: true})
	if err != nil {
		return IndexStatus{}, fmt.Errorf("get semantic index counts: %w", err)
	}
	running := false
	lastError := ""
	if s.backfilling != nil {
		backfillStatus := s.backfilling.Status()
		running = backfillStatus.Running
		lastError = backfillStatus.LastError
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
	case lastError != "":
		status.State = IndexStateUnavailable
		status.Message = lastError
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
