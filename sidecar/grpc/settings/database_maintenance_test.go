package settings

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDatabaseMaintenanceStatusReportsSqliteFootprint(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dsn := db.NewDSN(filepath.Join(dir, "maintenance.db"))
	writeFileOfSize(t, dsn.GetPath(), 11)
	writeFileOfSize(t, dsn.GetPath()+"-wal", 13)
	writeFileOfSize(t, dsn.GetPath()+"-shm", 17)

	server := &Server{database: &db.Database{DSN: dsn}}
	status, err := server.GetDatabaseMaintenanceStatus(ctx, &settingsv1.GetDatabaseMaintenanceStatusRequest{})
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if got, want := status.GetSizeBytes(), int64(41); got != want {
		t.Fatalf("expected size %d, got %d", want, got)
	}
}

func TestPruneDatabaseRangeValidatesWindow(t *testing.T) {
	ctx := context.Background()
	server, _, cleanup := newTestSettingsServer(t, ctx)
	defer cleanup()
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)

	_, err := server.PruneDatabaseRange(ctx, &settingsv1.PruneDatabaseRangeRequest{
		StartedAt: timestamppb.New(now),
		EndedAt:   timestamppb.New(now),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestPruneDatabaseRangeDeletesWholeRangeAndLinkedRows(t *testing.T) {
	ctx := context.Background()
	server, database, cleanup := newTestSettingsServer(t, ctx)
	defer cleanup()
	q := database.WriteQuerier
	dayStart := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	outStart := dayStart.Add(48 * time.Hour)

	inEventID := createTestTransitionEvent(t, ctx, q, "In Range App", dayStart.Add(9*time.Hour), dayStart.Add(10*time.Hour))
	outEventID := createTestTransitionEvent(t, ctx, q, "Out Range App", outStart.Add(9*time.Hour), outStart.Add(10*time.Hour))
	if _, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{Name: "Already Orphaned App"}); err != nil {
		t.Fatalf("upsert orphan application: %v", err)
	}
	// Semantic documents cascade-delete only through their timeline; "summary:in" (no timeline) and
	// "event:out" (out-of-range timeline) survive a prune of the in-range window.
	createTestSemanticDocument(t, ctx, q, "event:in", sql.NullInt64{Int64: inEventID, Valid: true})
	createTestSemanticDocument(t, ctx, q, "summary:in", sql.NullInt64{})
	createTestSemanticDocument(t, ctx, q, "event:out", sql.NullInt64{Int64: outEventID, Valid: true})

	createTestLedgerItem(t, ctx, q, "entry-in", dayStart.Add(time.Hour), []int64{inEventID})
	createTestLedgerItem(t, ctx, q, "entry-out", outStart.Add(time.Hour), []int64{inEventID, outEventID})

	response, err := server.PruneDatabaseRange(ctx, &settingsv1.PruneDatabaseRangeRequest{
		StartedAt: timestamppb.New(dayStart),
		EndedAt:   timestamppb.New(dayEnd),
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	// The in-range window prunes the two foreground processes of the in-range event; the timeline,
	// its semantic document and embedding, its metadata, the in-range ledger item, and the now-orphan
	// application cascade away. The out-of-range event/documents and the pre-existing orphan app remain.
	assertPruneCount(t, "ledger items", response.GetLedgerItemsDeleted(), 1)
	assertPruneCount(t, "ledger item timeline entries", response.GetLedgerItemTimelineEntriesDeleted(), 2)
	assertPruneCount(t, "timeline", response.GetTimelineDeleted(), 1)
	assertPruneCount(t, "foreground processes", response.GetForegroundProcessesDeleted(), 2)
	assertPruneCount(t, "foreground process metadata", response.GetForegroundProcessMetadataDeleted(), 1)
	assertPruneCount(t, "timeline semantic documents", response.GetTimelineSemanticDocumentsDeleted(), 1)
	assertPruneCount(t, "timeline embeddings", response.GetTimelineEmbeddingsDeleted(), 1)
	assertPruneCount(t, "applications", response.GetApplicationsDeleted(), 1)
	if response.GetSizeBytes() <= 0 {
		t.Fatalf("expected size bytes in prune response, got %d", response.GetSizeBytes())
	}

	assertTableCount(t, ctx, database.ReadConn, "foreground_processes", 2)
	assertTableCount(t, ctx, database.ReadConn, "foreground_process_metadata", 1)
	assertTableCount(t, ctx, database.ReadConn, "timeline", 1)
	assertTableCount(t, ctx, database.ReadConn, "timeline_semantic_documents", 2)
	assertTableCount(t, ctx, database.ReadConn, "timeline_embeddings", 2)
	assertTableCount(t, ctx, database.ReadConn, "ledger_items", 1)
	assertTableCount(t, ctx, database.ReadConn, "applications", 2)
	assertTableCount(t, ctx, database.ReadConn, "ledger_item_timeline_entries", 1)
}

func TestVacuumDatabaseCompactsSqliteFootprint(t *testing.T) {
	ctx := context.Background()
	server, database, cleanup := newTestSettingsServer(t, ctx)
	defer cleanup()

	if err := database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		if _, err := conn.ExecContext(ctx, `
CREATE TABLE vacuum_payload (
  id INTEGER PRIMARY KEY,
  payload BLOB NOT NULL
)`); err != nil {
			return fmt.Errorf("create payload table: %w", err)
		}
		for range 512 {
			if _, err := conn.ExecContext(ctx, "INSERT INTO vacuum_payload (payload) VALUES (zeroblob(4096))"); err != nil {
				return fmt.Errorf("insert payload: %w", err)
			}
		}
		return runWalCheckpointTruncate(ctx, conn)
	}); err != nil {
		t.Fatalf("seed vacuum payload: %v", err)
	}

	sizeWithRows, err := sqliteFootprintSize(database.DSN.GetPath())
	if err != nil {
		t.Fatalf("measure size with rows: %v", err)
	}
	if err := database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		_, err := conn.ExecContext(ctx, "DELETE FROM vacuum_payload")
		return err
	}); err != nil {
		t.Fatalf("delete payload: %v", err)
	}
	sizeAfterDelete, err := sqliteFootprintSize(database.DSN.GetPath())
	if err != nil {
		t.Fatalf("measure size after delete: %v", err)
	}
	mainInfo, err := os.Stat(sqliteDatabasePath(database.DSN.GetPath()))
	if err != nil {
		t.Fatalf("stat main database: %v", err)
	}
	if sizeAfterDelete <= mainInfo.Size() {
		t.Fatalf("expected WAL/SHM bytes in footprint, got footprint %d and main file %d", sizeAfterDelete, mainInfo.Size())
	}

	response, err := server.VacuumDatabase(ctx, &settingsv1.VacuumDatabaseRequest{})
	if err != nil {
		t.Fatalf("vacuum: %v", err)
	}
	if got, want := response.GetSizeBeforeBytes(), sizeAfterDelete; got != want {
		t.Fatalf("expected size before %d, got %d", want, got)
	}
	if response.GetSizeAfterBytes() >= response.GetSizeBeforeBytes() {
		t.Fatalf("expected vacuum to reduce footprint from %d, got %d", response.GetSizeBeforeBytes(), response.GetSizeAfterBytes())
	}
	if response.GetSizeAfterBytes() >= sizeWithRows {
		t.Fatalf("expected vacuumed footprint %d to be smaller than populated footprint %d", response.GetSizeAfterBytes(), sizeWithRows)
	}
	walInfo, err := os.Stat(sqliteDatabasePath(database.DSN.GetPath()) + "-wal")
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat WAL: %v", err)
	}
	if err == nil && walInfo.Size() != 0 {
		t.Fatalf("expected truncated WAL, got %d bytes", walInfo.Size())
	}
}

func newTestSettingsServer(t *testing.T, ctx context.Context) (*Server, *db.Database, func()) {
	t.Helper()

	dsn := db.NewDSN(filepath.Join(t.TempDir(), "settings.db"))
	dsn.SetJournalMode(enumsjournalmode.WAL)
	dsn.EnableFK()
	dsn.SetBusyTimeout(5 * time.Second)
	database, err := db.New(ctx, db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	registry := services.New()
	db.Register(registry, database)
	server := NewServer(registry)

	return server, database, func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	}
}

// createTestTransitionEvent inserts an event into the event store (two foreground_processes boundary
// rows tied together by a timeline row) and returns the timeline id, which plays the role of the old
// transition-event id.
func createTestTransitionEvent(t *testing.T, ctx context.Context, q writequeries.Querier, appName string, startedAt time.Time, endedAt time.Time) int64 {
	t.Helper()

	appID, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
		Name: appName,
	})
	if err != nil {
		t.Fatalf("upsert application: %v", err)
	}
	application := sql.NullInt64{Int64: appID, Valid: true}

	initialFP, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
		ApplicationID: application,
		Pid:           4242,
		CreatedAtUtc:  startedAt.UTC(),
	})
	if err != nil {
		t.Fatalf("upsert initial foreground process: %v", err)
	}
	if err := q.CreateForegroundProcessMetadata(ctx, writequeries.CreateForegroundProcessMetadataParams{
		ForegroundProcessID: initialFP,
		Browser:             false,
		Idle:                false,
	}); err != nil {
		t.Fatalf("create foreground process metadata: %v", err)
	}
	endFP, err := q.UpsertForegroundProcess(ctx, writequeries.UpsertForegroundProcessParams{
		ApplicationID: application,
		Pid:           4242,
		CreatedAtUtc:  endedAt.UTC(),
	})
	if err != nil {
		t.Fatalf("upsert end foreground process: %v", err)
	}
	timelineID, err := q.CreateTimeline(ctx, writequeries.CreateTimelineParams{
		InitialForegroundProcessID: sql.NullInt64{Int64: initialFP, Valid: true},
		EndForegroundProcessID:     sql.NullInt64{Int64: endFP, Valid: true},
	})
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	return timelineID
}

func createTestSemanticDocument(t *testing.T, ctx context.Context, q writequeries.Querier, key string, timelineID sql.NullInt64) {
	t.Helper()

	documentID, err := q.UpsertTimelineSemanticDocument(ctx, writequeries.UpsertTimelineSemanticDocumentParams{
		DocumentKey: key,
		TimelineID:  timelineID,
		Type:        sql.NullInt64{Int64: 1, Valid: true},
		Content:     key,
	})
	if err != nil {
		t.Fatalf("upsert semantic document: %v", err)
	}
	vector, err := semantic.NormalizeFloat32Vector([]float32{1}, semantic.StoreEmbeddingDimension)
	if err != nil {
		t.Fatalf("normalize embedding: %v", err)
	}
	encoded, err := semantic.EncodeFloat32Vector(vector)
	if err != nil {
		t.Fatalf("encode embedding: %v", err)
	}
	if err := q.CreateTimelineEmbedding(ctx, writequeries.CreateTimelineEmbeddingParams{
		TimelineID:                  timelineID,
		TimelineSemanticDocumentsID: sql.NullInt64{Int64: documentID, Valid: true},
		EmbeddingModelID:            sql.NullInt64{Int64: 1, Valid: true},
		Dimension:                   semantic.StoreEmbeddingDimension,
		Embedding:                   encoded,
	}); err != nil {
		t.Fatalf("create semantic embedding: %v", err)
	}
}

// createTestLedgerItem inserts a ledger item (started at startedAt) and links it to the given timeline
// ids via ledger_item_timeline_entries (the successor to the old usage blocks).
func createTestLedgerItem(t *testing.T, ctx context.Context, q writequeries.Querier, title string, startedAt time.Time, timelineIDs []int64) {
	t.Helper()

	// The ledger is not seeded; it is created lazily before the first ledger item.
	if err := q.EnsureLedger(ctx); err != nil {
		t.Fatalf("ensure ledger: %v", err)
	}
	item, err := q.CreateLedgerItem(ctx, writequeries.CreateLedgerItemParams{
		Billable:     true,
		Title:        title,
		Notes:        sql.NullString{},
		StartedAtUtc: sql.NullTime{Time: startedAt.UTC(), Valid: true},
		EndedAtUtc:   sql.NullTime{Time: startedAt.Add(30 * time.Minute).UTC(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create ledger item: %v", err)
	}
	for _, timelineID := range timelineIDs {
		if err := q.CreateLedgerItemTimelineEntry(ctx, writequeries.CreateLedgerItemTimelineEntryParams{
			LedgerItemsID: sql.NullInt64{Int64: item.ID, Valid: true},
			TimelineID:    sql.NullInt64{Int64: timelineID, Valid: true},
		}); err != nil {
			t.Fatalf("create ledger item timeline entry: %v", err)
		}
	}
}

func assertTableCount(t *testing.T, ctx context.Context, conn *sql.DB, table string, want int64) {
	t.Helper()

	var got int64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("expected %s count %d, got %d", table, want, got)
	}
}

func assertPruneCount(t *testing.T, name string, got int64, want int64) {
	t.Helper()
	if got != want {
		t.Fatalf("expected %s deleted %d, got %d", name, want, got)
	}
}

func writeFileOfSize(t *testing.T, path string, size int) {
	t.Helper()

	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
