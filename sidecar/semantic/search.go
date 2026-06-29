package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type Searcher struct {
	queries  *queries.Queries
	embedder Embedder
	clock    func() time.Time
	location *time.Location
}

type SearchResult struct {
	DocumentID        int64
	DocumentKey       string
	DocumentType      string
	TransitionEventID int64
	StartedAt         time.Time
	EndedAt           time.Time
	Content           string
	Distance          float64
}

func NewSearcher(conn queries.DBTX, embedder Embedder) *Searcher {
	return &Searcher{
		queries:  queries.New(conn),
		embedder: embedder,
		clock:    time.Now,
		location: time.Local,
	}
}

func (s *Searcher) Search(ctx context.Context, query string, k int64) ([]SearchResult, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive")
	}

	expandedQuery := ExpandSemanticQuery(query, s.now(), s.location)
	embedding, err := s.embedder.Embed(ctx, expandedQuery)
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

	candidateCount := candidateSourceCount(k)
	rows, err := s.queries.SearchSemanticDocuments(ctx, queries.SearchSemanticDocumentsParams{
		Embedding:      encoded,
		EmbeddingModel: s.embedder.Model(),
		K:              sql.NullInt64{Int64: candidateCount, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("search semantic documents: %w", err)
	}

	results := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		transitionEventID := int64(0)
		if row.TransitionEventID.Valid {
			transitionEventID = row.TransitionEventID.Int64
		}
		startedAt := time.Time{}
		if row.StartedAt.Valid {
			startedAt = row.StartedAt.Time
		}
		endedAt := time.Time{}
		if row.EndedAt.Valid {
			endedAt = row.EndedAt.Time
		}
		results = append(results, SearchResult{
			DocumentID:        row.ID,
			DocumentKey:       row.DocumentKey,
			DocumentType:      row.DocumentType,
			TransitionEventID: transitionEventID,
			StartedAt:         startedAt,
			EndedAt:           endedAt,
			Content:           row.Content,
			Distance:          row.Distance.Float64,
		})
	}

	return diversifySearchResults(query, results, int(k)), nil
}

func (s *Searcher) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock()
}

func candidateSourceCount(k int64) int64 {
	if k <= 0 {
		return 10
	}
	count := k * 5
	if count < 20 {
		count = 20
	}
	if count > 50 {
		count = 50
	}
	return count
}

func diversifySearchResults(query string, candidates []SearchResult, limit int) []SearchResult {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}

	typeOrder := preferredDocumentTypeOrder(query)
	selected := make([]SearchResult, 0, limit)
	used := map[string]struct{}{}
	for _, documentType := range typeOrder {
		for _, candidate := range candidates {
			if candidate.DocumentType != documentType {
				continue
			}
			if _, ok := used[candidate.DocumentKey]; ok {
				continue
			}
			selected = append(selected, candidate)
			used[candidate.DocumentKey] = struct{}{}
			if len(selected) == limit {
				return selected
			}
			break
		}
	}

	for _, candidate := range candidates {
		if _, ok := used[candidate.DocumentKey]; ok {
			continue
		}
		selected = append(selected, candidate)
		if len(selected) == limit {
			return selected
		}
	}

	return selected
}

func preferredDocumentTypeOrder(query string) []string {
	normalized := strings.ToLower(query)
	if strings.Contains(normalized, "today") ||
		strings.Contains(normalized, "yesterday") ||
		strings.Contains(normalized, "morning") ||
		strings.Contains(normalized, "afternoon") ||
		strings.Contains(normalized, "evening") ||
		strings.Contains(normalized, "throughout") ||
		strings.Contains(normalized, "spend time") ||
		strings.Contains(normalized, "spent time") ||
		strings.Contains(normalized, "day") {
		return []string{DocumentTypeDaySummary, DocumentTypeTimeBlock, DocumentTypeAppDay, DocumentTypeEvent}
	}
	return []string{DocumentTypeEvent, DocumentTypeAppDay, DocumentTypeDaySummary, DocumentTypeTimeBlock}
}
