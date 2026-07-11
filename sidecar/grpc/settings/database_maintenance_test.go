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
	outEnd := outStart.Add(24 * time.Hour)

	inEventID := createTestTransitionEvent(t, ctx, q, "In Range App", dayStart.Add(9*time.Hour), dayStart.Add(10*time.Hour))
	outEventID := createTestTransitionEvent(t, ctx, q, "Out Range App", outStart.Add(9*time.Hour), outStart.Add(10*time.Hour))
	if _, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{Name: "Already Orphaned App"}); err != nil {
		t.Fatalf("upsert orphan application: %v", err)
	}
	createTestSemanticDocument(t, ctx, q, "event:in", sql.NullInt64{Int64: inEventID, Valid: true}, dayStart.Add(9*time.Hour), dayStart.Add(10*time.Hour))
	createTestSemanticDocument(t, ctx, q, "summary:in", sql.NullInt64{}, dayStart.Add(11*time.Hour), dayStart.Add(12*time.Hour))
	createTestSemanticDocument(t, ctx, q, "event:out", sql.NullInt64{Int64: outEventID, Valid: true}, outStart.Add(9*time.Hour), outStart.Add(10*time.Hour))

	createTestTimesheetEntry(t, ctx, q, "sheet-in", "entry-in", dayStart, dayEnd, []string{
		fmt.Sprintf("sidecar-%d", inEventID),
	})
	createTestTimesheetEntry(t, ctx, q, "sheet-out", "entry-out", outStart, outEnd, []string{
		fmt.Sprintf("sidecar-%d", inEventID),
		fmt.Sprintf("sidecar-%d", outEventID),
	})

	response, err := server.PruneDatabaseRange(ctx, &settingsv1.PruneDatabaseRangeRequest{
		StartedAt: timestamppb.New(dayStart),
		EndedAt:   timestamppb.New(dayEnd),
	})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	assertPruneCount(t, "timesheet entries", response.GetTimesheetEntriesDeleted(), 1)
	assertPruneCount(t, "usage links", response.GetUsageLinksDeleted(), 2)
	assertPruneCount(t, "timesheets", response.GetTimesheetsDeleted(), 1)
	assertPruneCount(t, "transition events", response.GetTransitionEventsDeleted(), 1)
	assertPruneCount(t, "transition metadata", response.GetTransitionMetadataDeleted(), 1)
	assertPruneCount(t, "semantic documents", response.GetSemanticDocumentsDeleted(), 2)
	assertPruneCount(t, "embeddings", response.GetEmbeddingsDeleted(), 2)
	assertPruneCount(t, "applications", response.GetApplicationsDeleted(), 1)
	if response.GetSizeBytes() <= 0 {
		t.Fatalf("expected size bytes in prune response, got %d", response.GetSizeBytes())
	}

	assertTableCount(t, ctx, database.ReadConn, "transition_events", 1)
	assertTableCount(t, ctx, database.ReadConn, "transition_event_metadata", 1)
	assertTableCount(t, ctx, database.ReadConn, "semantic_documents", 1)
	assertTableCount(t, ctx, database.ReadConn, "semantic_document_embeddings", 1)
	assertTableCount(t, ctx, database.ReadConn, "timesheet_entries", 1)
	assertTableCount(t, ctx, database.ReadConn, "timesheets", 1)
	assertTableCount(t, ctx, database.ReadConn, "applications", 2)
	assertTableCount(t, ctx, database.ReadConn, "timesheet_entry_usage_blocks", 1)
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

func createTestTransitionEvent(t *testing.T, ctx context.Context, q writequeries.Querier, appName string, startedAt time.Time, endedAt time.Time) int64 {
	t.Helper()

	appID, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
		Name: appName,
	})
	if err != nil {
		t.Fatalf("upsert application: %v", err)
	}
	eventID, err := q.CreateTransitionEvent(ctx, writequeries.CreateTransitionEventParams{
		ApplicationID: sql.NullInt64{Int64: appID, Valid: true},
		Reason:        "focus_change",
		StartedAt:     startedAt,
		EndedAt:       endedAt,
	})
	if err != nil {
		t.Fatalf("create transition event: %v", err)
	}
	if err := q.CreateTransitionEventMetadata(ctx, writequeries.CreateTransitionEventMetadataParams{
		TransitionEventID: eventID,
		Browser:           false,
		Idle:              false,
	}); err != nil {
		t.Fatalf("create transition metadata: %v", err)
	}
	return eventID
}

func createTestSemanticDocument(t *testing.T, ctx context.Context, q writequeries.Querier, key string, eventID sql.NullInt64, startedAt time.Time, endedAt time.Time) {
	t.Helper()

	documentID, err := q.UpsertSemanticDocument(ctx, writequeries.UpsertSemanticDocumentParams{
		DocumentKey:       key,
		DocumentType:      "event",
		TransitionEventID: eventID,
		StartedAt:         sql.NullTime{Time: startedAt, Valid: true},
		EndedAt:           sql.NullTime{Time: endedAt, Valid: true},
		Content:           key,
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
	if err := q.CreateSemanticDocumentEmbedding(ctx, writequeries.CreateSemanticDocumentEmbeddingParams{
		SemanticDocumentID: documentID,
		EmbeddingModel:     "test-model",
		EmbeddingDimension: semantic.StoreEmbeddingDimension,
		EmbeddedAt:         sql.NullTime{Time: startedAt, Valid: true},
		Embedding:          encoded,
	}); err != nil {
		t.Fatalf("create semantic embedding: %v", err)
	}
}

func createTestTimesheetEntry(t *testing.T, ctx context.Context, q writequeries.Querier, sheetID string, entryID string, startedAt time.Time, endedAt time.Time, usageIDs []string) {
	t.Helper()

	now := time.Now().UTC()
	if _, err := q.EnsureTimesheet(ctx, writequeries.EnsureTimesheetParams{
		ID:        sheetID,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create timesheet: %v", err)
	}
	if _, err := q.CreateTimesheetEntry(ctx, writequeries.CreateTimesheetEntryParams{
		ID:              entryID,
		TimesheetID:     sheetID,
		Title:           entryID,
		Notes:           "",
		StartMinute:     60,
		DurationMinutes: 30,
		Billable:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create timesheet entry: %v", err)
	}
	for index, usageID := range usageIDs {
		if err := q.CreateTimesheetEntryUsageBlock(ctx, writequeries.CreateTimesheetEntryUsageBlockParams{
			TimesheetEntryID: entryID,
			UsageID:          usageID,
			SortOrder:        int64(index),
		}); err != nil {
			t.Fatalf("create usage block: %v", err)
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
