package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Searcher struct {
	conn        *sql.DB
	vectorStore *SQLiteVectorStore
	embedder    Embedder
	clock       func() time.Time
	location    *time.Location
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

func NewSearcher(conn *sql.DB, embedder Embedder, vectorStore ...*SQLiteVectorStore) *Searcher {
	store := NewSQLiteVectorStore(conn)
	if len(vectorStore) > 0 && vectorStore[0] != nil {
		store = vectorStore[0]
	}
	return &Searcher{
		conn:        conn,
		vectorStore: store,
		embedder:    embedder,
		clock:       time.Now,
		location:    time.Local,
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

	if s.vectorStore == nil {
		return nil, fmt.Errorf("sqlite-vector store is required")
	}
	hasEmbeddings, err := s.hasEmbeddings(ctx, s.embedder.Model())
	if err != nil {
		return nil, err
	}
	if !hasEmbeddings {
		return nil, nil
	}
	if err := s.vectorStore.EnsureQuantized(ctx); err != nil {
		return nil, fmt.Errorf("prepare sqlite-vector semantic search: %w", err)
	}

	candidateCount := candidateSourceCount(k)
	rows, err := s.conn.QueryContext(ctx, searchSemanticDocumentsSQL, encoded, s.embedder.Model(), candidateCount)
	if err != nil {
		return nil, fmt.Errorf("search semantic documents: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var row struct {
			ID                int64
			DocumentKey       string
			DocumentType      string
			TransitionEventID sql.NullInt64
			StartedAt         sql.NullTime
			EndedAt           sql.NullTime
			Content           string
			Distance          sql.NullFloat64
		}
		if err := rows.Scan(
			&row.ID,
			&row.DocumentKey,
			&row.DocumentType,
			&row.TransitionEventID,
			&row.StartedAt,
			&row.EndedAt,
			&row.Content,
			&row.Distance,
		); err != nil {
			return nil, fmt.Errorf("scan semantic search result: %w", err)
		}
		transitionEventID := int64(0)
		if row.TransitionEventID.Valid {
			transitionEventID = row.TransitionEventID.Int64
		}
		startedAt := time.Time{}
		if row.StartedAt.Valid {
			startedAt = row.StartedAt.Time.UTC()
		}
		endedAt := time.Time{}
		if row.EndedAt.Valid {
			endedAt = row.EndedAt.Time.UTC()
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read semantic search results: %w", err)
	}

	return diversifySearchResults(query, results, int(k)), nil
}

const searchSemanticDocumentsSQL = `
SELECT
  semantic_documents.id,
  semantic_documents.document_key,
  semantic_documents.document_type,
  semantic_documents.transition_event_id,
  semantic_documents.started_at,
  semantic_documents.ended_at,
  semantic_documents.content,
  vector_matches.distance
FROM vector_quantize_scan('semantic_document_embeddings', 'embedding', ?) AS vector_matches
JOIN semantic_document_embeddings
  ON semantic_document_embeddings.rowid = vector_matches.rowid
JOIN semantic_documents
  ON semantic_documents.id = semantic_document_embeddings.semantic_document_id
WHERE semantic_document_embeddings.embedding_model = ?
ORDER BY vector_matches.distance
LIMIT ?`

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

func (s *Searcher) hasEmbeddings(ctx context.Context, embeddingModel string) (bool, error) {
	var count int64
	if err := s.conn.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM semantic_document_embeddings
WHERE embedding_model = ?`, embeddingModel).Scan(&count); err != nil {
		return false, fmt.Errorf("count semantic embeddings: %w", err)
	}
	return count > 0, nil
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
