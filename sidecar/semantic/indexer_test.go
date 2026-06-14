package semantic

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Model() string { return "fake-embedding" }

func (fakeEmbedder) Dimension() int { return 2048 }

func (fakeEmbedder) Embed(_ context.Context, input string) ([]float32, error) {
	values := make([]float32, 2048)
	if strings.Contains(input, "Google Chrome") || strings.Contains(input, "browser") {
		values[0] = 1
	} else {
		values[1] = 1
	}
	return values, nil
}

func TestIndexerAndSearcher(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	eventID := createSemanticTestTransitionEvent(t, ctx, database.Conn)

	indexer := NewIndexer(database.Conn, fakeEmbedder{})
	documentID, err := indexer.IndexTransitionEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("index transition event: %v", err)
	}
	if documentID == 0 {
		t.Fatal("expected document id")
	}

	searcher := NewSearcher(database.Conn, fakeEmbedder{})
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
	if !strings.Contains(results[0].Content, "Application: Google Chrome") {
		t.Fatalf("unexpected content:\n%s", results[0].Content)
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
	q := queries.New(conn)
	applicationID, err := q.UpsertApplication(ctx, "Google Chrome")
	if err != nil {
		t.Fatalf("upsert application: %v", err)
	}

	eventID, err := q.CreateTransitionEvent(ctx, queries.CreateTransitionEventParams{
		ApplicationID: sql.NullInt64{Int64: applicationID, Valid: true},
		Reason:        "tab_change",
		StartedAt:     time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		EndedAt:       time.Date(2026, 6, 13, 10, 5, 0, 0, time.UTC),
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
