package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	sqlitevector "github.com/skulpturenz/timeboxxing/sidecar/db/sqlite-vector"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
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
	database := newSemanticVectorTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteQuerier)

	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1)
	documentID, err := indexer.IndexTransitionEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("index transition event: %v", err)
	}
	if documentID == 0 {
		t.Fatal("expected document id")
	}

	searcher := NewSearcher(database.ReadConn, fakeEmbedder{}, 1)
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
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteQuerier)
	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1)

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

	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "timeline_semantic_documents", "type = 1"); count != 1 {
		t.Fatalf("expected one semantic event document, got %d", count)
	}
	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "timeline_embeddings", "timeline_semantic_documents_id IN (SELECT id FROM timeline_semantic_documents WHERE type = 1)"); count != 1 {
		t.Fatalf("expected one semantic event embedding, got %d", count)
	}
}

func TestIndexerDoesNotPersistOnEmbeddingFailure(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteQuerier)
	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, failingEmbedder{}, 1)

	if _, err := indexer.IndexTransitionEvent(ctx, eventID); err == nil {
		t.Fatal("expected embedding error")
	}
	if count := countSemanticRows(t, ctx, database.ReadConn, "timeline_semantic_documents"); count != 0 {
		t.Fatalf("expected no semantic documents, got %d", count)
	}
	if count := countSemanticRows(t, ctx, database.ReadConn, "timeline_embeddings"); count != 0 {
		t.Fatalf("expected no semantic embeddings, got %d", count)
	}
}

// An observation recorded before the ingest normalised to UTC is stored with the monitor's local
// offset, while the day window is asked for in UTC. Regression: compared as text, an event whose
// local wall clock landed on the following day read as outside the window and dropped out of the
// day's summaries.
func TestIndexerSummarisesEventsStoredWithANonUTCOffset(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	// late enough that a +12:00 wall clock reads as the next day
	startedAt := time.Date(2026, 6, 13, 20, 0, 0, 0, time.UTC)
	createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, startedAt)
	restoreStoredOffset(t, database, 12)

	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1)
	// pin the day boundary so the window does not depend on the machine's zone
	indexer.location = time.UTC

	if err := indexer.RefreshSummariesForTime(ctx, startedAt); err != nil {
		t.Fatalf("refresh summaries: %v", err)
	}

	count := countSemanticRowsWhere(t, ctx, database.ReadConn, "timeline_semantic_documents",
		"document_key LIKE 'app_day:%' AND content LIKE '%Google Chrome%'")
	if count != 1 {
		t.Fatalf("expected the event to be summarised, got %d app-day documents", count)
	}
}

// restoreStoredOffset rewrites every stored observation as the same instant expressed at a fixed
// offset from UTC, reproducing the rows written before the ingest normalised to UTC.
func restoreStoredOffset(t *testing.T, database *db.Database, offsetHours int) {
	t.Helper()

	if err := database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		_, err := conn.Exec(
			`UPDATE foreground_processes
			 SET created_at_utc = strftime('%Y-%m-%d %H:%M:%f', created_at_utc, ?) || ?`,
			fmt.Sprintf("%+d hours", offsetHours),
			fmt.Sprintf("%+03d:00", offsetHours),
		)

		return err
	}); err != nil {
		t.Fatalf("restore stored offset: %v", err)
	}
}

func TestIndexerValidationFailures(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.WriteQuerier)

	if _, err := NewIndexer(database.WriteQuerier, database.ReadQuerier, dimensionMismatchEmbedder{}, 1).IndexTransitionEvent(ctx, eventID); err == nil {
		t.Fatal("expected dimension mismatch error")
	}
	if _, err := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1).IndexTransitionEvent(ctx, eventID+1000); err == nil {
		t.Fatal("expected missing source error")
	}
}

func newSemanticTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	return newSemanticTestDatabaseWithSQLiteVector(t, ctx, "")
}

func newSemanticVectorTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	extensionPath := os.Getenv("SIDECAR_SQLITE_VECTOR_EXTENSION_PATH")
	options := sqlitevector.Options{}
	if strings.TrimSpace(extensionPath) != "" {
		options.Path = &extensionPath
	}
	if _, _, err := options.Load(); err != nil {
		t.Skip("sqlite-vector extension is not bundled for this platform and SIDECAR_SQLITE_VECTOR_EXTENSION_PATH is not set")
	}
	return newSemanticTestDatabaseWithSQLiteVector(t, ctx, extensionPath)
}

func newSemanticTestDatabaseWithSQLiteVector(t *testing.T, ctx context.Context, extensionPath string) *db.Database {
	t.Helper()
	var extensionPathOption *string
	if strings.TrimSpace(extensionPath) != "" {
		extensionPathOption = &extensionPath
	}
	database, err := db.New(ctx, db.Options{
		DSN:                       db.NewDSN(filepath.Join(t.TempDir(), "test.db")),
		SQLiteVectorExtensionPath: extensionPathOption,
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

func createSemanticTestTransitionEvent(t *testing.T, ctx context.Context, q writequeries.Querier) int64 {
	t.Helper()
	return createSemanticTestTransitionEventAt(t, ctx, q, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
}

func ptr[T any](v T) *T { return &v }

func createSemanticTestTransitionEventAt(t *testing.T, ctx context.Context, q writequeries.Querier, startedAt time.Time) int64 {
	t.Helper()
	applicationID, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
		Name: "Google Chrome",
	})
	if err != nil {
		t.Fatalf("upsert application: %v", err)
	}

	application := &applicationID

	initialFP, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
		ApplicationID: application,
		Pid:           ptr(int64(4242)),
		CreatedAtUtc:  startedAt.UTC(),
	})
	if err != nil {
		t.Fatalf("upsert initial foreground process: %v", err)
	}
	tab := "GitHub"
	url := "https://github.com/"
	if _, err := q.InsertForegroundProcessMetadata(ctx, writequeries.InsertForegroundProcessMetadataParams{
		ForegroundProcessID: initialFP,
		Browser:             true,
		Idle:                false,
		Tab:                 &tab,
		CdpUrl:              &url,
	}); err != nil {
		t.Fatalf("insert foreground process metadata: %v", err)
	}
	endFP, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
		ApplicationID: application,
		Pid:           ptr(int64(4242)),
		CreatedAtUtc:  startedAt.Add(5 * time.Minute).UTC(),
	})
	if err != nil {
		t.Fatalf("upsert end foreground process: %v", err)
	}
	timelineID, err := q.UpsertTimeline(ctx, writequeries.UpsertTimelineParams{
		InitialForegroundProcessID: &initialFP,
		EndForegroundProcessID:     &endFP,
	})
	if err != nil {
		t.Fatalf("upsert timeline: %v", err)
	}

	return timelineID
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
