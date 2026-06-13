package semantic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type Searcher struct {
	queries  *queries.Queries
	embedder Embedder
}

type SearchResult struct {
	TransitionEventID int64
	Content           string
	Distance          float64
}

func NewSearcher(conn queries.DBTX, embedder Embedder) *Searcher {
	return &Searcher{queries: queries.New(conn), embedder: embedder}
}

func (s *Searcher) Search(ctx context.Context, query string, k int64) ([]SearchResult, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive")
	}

	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed semantic search query: %w", err)
	}
	if len(embedding) != s.embedder.Dimension() {
		return nil, fmt.Errorf("embedding dimension mismatch: got %d, want %d", len(embedding), s.embedder.Dimension())
	}
	encoded, err := EncodeFloat32Vector(embedding)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.SearchTransitionEventDocuments(ctx, queries.SearchTransitionEventDocumentsParams{
		Embedding: encoded,
		K:         sql.NullInt64{Int64: k, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("search transition event documents: %w", err)
	}

	results := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, SearchResult{
			TransitionEventID: row.TransitionEventID,
			Content:           row.Content,
			Distance:          row.Distance.Float64,
		})
	}

	return results, nil
}
