package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Model() string { return "fake-embedding" }

func (fakeEmbedder) Dimension() int { return StoreEmbeddingDimension }

func (fakeEmbedder) Embed(_ context.Context, input string) ([]float32, error) {
	values := make([]float32, StoreEmbeddingDimension)
	if strings.Contains(input, "Google Chrome") || strings.Contains(input, "browser") {
		values[0] = 1
	} else {
		values[1] = 1
	}
	return values, nil
}

type dimensionMismatchEmbedder struct{}

func (dimensionMismatchEmbedder) Model() string { return "bad-embedding" }

func (dimensionMismatchEmbedder) Dimension() int { return StoreEmbeddingDimension }

func (dimensionMismatchEmbedder) Embed(context.Context, string) ([]float32, error) {
	return make([]float32, 3), nil
}

type failingEmbedder struct{}

func (failingEmbedder) Model() string { return "failing-embedding" }

func (failingEmbedder) Dimension() int { return StoreEmbeddingDimension }

func (failingEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, fmt.Errorf("embed failed")
}

func TestIndexerAndSearcher(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteConn)

	indexer := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{})
	documentID, err := indexer.IndexTransitionEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("index transition event: %v", err)
	}
	if documentID == 0 {
		t.Fatal("expected document id")
	}

	searcher := NewSearcher(database.ReadConn, fakeEmbedder{})
	results, err := searcher.Search(ctx, "browser work", 1)
	if err != nil {
		t.Fatalf("search transition event documents: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].TransitionEventID != eventID {
		t.Fatalf("expected transition event id %d, got %d", eventID, results[0].TransitionEventID)
	}
	if results[0].DocumentType != DocumentTypeEvent {
		t.Fatalf("expected event document, got %q", results[0].DocumentType)
	}
	if !strings.Contains(results[0].Content, "Application: Google Chrome") {
		t.Fatalf("unexpected content:\n%s", results[0].Content)
	}
}

func TestIndexerIsIdempotent(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteConn)
	indexer := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{})

	firstDocumentID, err := indexer.IndexTransitionEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("index transition event first time: %v", err)
	}
	secondDocumentID, err := indexer.IndexTransitionEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("index transition event second time: %v", err)
	}
	if secondDocumentID != firstDocumentID {
		t.Fatalf("expected same document id, got first=%d second=%d", firstDocumentID, secondDocumentID)
	}

	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "semantic_documents", "document_type = 'event'"); count != 1 {
		t.Fatalf("expected one semantic event document, got %d", count)
	}
	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "semantic_document_float32_embeddings", "semantic_document_id IN (SELECT id FROM semantic_documents WHERE document_type = 'event')"); count != 1 {
		t.Fatalf("expected one semantic event embedding, got %d", count)
	}
}

func TestIndexerDoesNotPersistOnEmbeddingFailure(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteConn)
	indexer := NewIndexer(database.WriteConn, database.ReadQuerier, failingEmbedder{})

	if _, err := indexer.IndexTransitionEvent(ctx, eventID); err == nil {
		t.Fatal("expected embedding error")
	}
	if count := countSemanticRows(t, ctx, database.ReadConn, "semantic_documents"); count != 0 {
		t.Fatalf("expected no semantic documents, got %d", count)
	}
	if count := countSemanticRows(t, ctx, database.ReadConn, "semantic_document_float32_embeddings"); count != 0 {
		t.Fatalf("expected no semantic embeddings, got %d", count)
	}
}

func TestIndexerValidationFailures(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteConn)

	if _, err := NewIndexer(database.WriteConn, database.ReadQuerier, dimensionMismatchEmbedder{}).IndexTransitionEvent(ctx, eventID); err == nil {
		t.Fatal("expected dimension mismatch error")
	}
	if _, err := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{}).IndexTransitionEvent(ctx, eventID+1000); err == nil {
		t.Fatal("expected missing source error")
	}
}

func newSemanticTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	database, err := db.New(ctx, db.Options{
		Engine:         db.EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return database
}

func createSemanticTestTransitionEvent(t *testing.T, ctx context.Context, conn *sql.DB) int64 {
	t.Helper()
	return createSemanticTestTransitionEventAt(t, ctx, conn, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
}

func createSemanticTestTransitionEventAt(t *testing.T, ctx context.Context, conn *sql.DB, startedAt time.Time) int64 {
	t.Helper()
	q := queries.New(conn)
	applicationID, err := q.UpsertApplication(ctx, queries.UpsertApplicationParams{
		Name: "Google Chrome",
	})
	if err != nil {
		t.Fatalf("upsert application: %v", err)
	}

	eventID, err := q.CreateTransitionEvent(ctx, queries.CreateTransitionEventParams{
		ApplicationID: sql.NullInt64{Int64: applicationID, Valid: true},
		Reason:        "tab_change",
		StartedAt:     startedAt,
		EndedAt:       startedAt.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create transition event: %v", err)
	}
	tab := "GitHub"
	url := "https://github.com/"
	if err := q.CreateTransitionEventMetadata(ctx, queries.CreateTransitionEventMetadataParams{
		TransitionEventID: eventID,
		Browser:           true,
		Tab:               &tab,
		Idle:              false,
		CdpUrl:            &url,
	}); err != nil {
		t.Fatalf("create transition event metadata: %v", err)
	}

	return eventID
}

func countSemanticRows(t *testing.T, ctx context.Context, conn *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func countSemanticRowsWhere(t *testing.T, ctx context.Context, conn *sql.DB, table string, where string) int {
	t.Helper()
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+where).Scan(&count); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	return count
}
