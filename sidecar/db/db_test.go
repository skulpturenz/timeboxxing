package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

func TestSqliteVecIsAvailable(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	var version string
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT vec_version()`).Scan(&version); err != nil {
		t.Fatalf("query sqlite-vec version: %v", err)
	}
	if version == "" {
		t.Fatal("expected sqlite-vec version")
	}
}

func TestSqliteMigrationsRunOnce(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "test.db")

	database, err := New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: dsn,
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	database, err = New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: dsn,
	})
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	defer database.Close()

	assertMigrationTableVersion(t, ctx, database, "schema_migrations", 3)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_embedding_models", 1)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_semantic_models", 1)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_settings", 1)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_transition_event_reasons", 1)
}

func TestSqliteProjectsMigrationCreatesTable(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	if _, err := database.WriteQuerier.CreateProject(ctx, queries.CreateProjectParams{
		ID:        "client-work",
		Name:      "Client Work",
		ColorArgb: 0xFF00FFEE,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	projects, err := database.ReadQuerier.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Client != "" {
		t.Fatalf("expected empty client, got %q", projects[0].Client)
	}
	if projects[0].HourlyRateCents != 0 {
		t.Fatalf("expected zero hourly rate, got %d", projects[0].HourlyRateCents)
	}
}

func TestSqliteTimesheetsMigrationCreatesTables(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC()
	timesheet, err := database.WriteQuerier.CreateTimesheet(ctx, queries.CreateTimesheetParams{
		ID:        "timesheet-test",
		StartedAt: now,
		EndedAt:   now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create timesheet: %v", err)
	}
	entry, err := database.WriteQuerier.CreateTimesheetEntry(ctx, queries.CreateTimesheetEntryParams{
		ID:              "entry-test",
		TimesheetID:     timesheet.ID,
		Title:           "Design review",
		Notes:           "",
		StartMinute:     9 * 60,
		DurationMinutes: 30,
		Billable:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("create timesheet entry: %v", err)
	}
	if err := database.WriteQuerier.CreateTimesheetEntryUsageBlock(ctx, queries.CreateTimesheetEntryUsageBlockParams{
		TimesheetEntryID: entry.ID,
		UsageID:          "sidecar-123",
		SortOrder:        0,
	}); err != nil {
		t.Fatalf("create usage block: %v", err)
	}

	entries, err := database.ReadQuerier.ListTimesheetEntries(ctx, queries.ListTimesheetEntriesParams{
		StartedAt: timesheet.StartedAt,
		EndedAt:   timesheet.EndedAt,
	})
	if err != nil {
		t.Fatalf("list timesheet entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 timesheet entry, got %d", len(entries))
	}
}

func TestSqliteUsesWALAndSingleWriterPool(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		Engine:         EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	if database.WriteConn == nil {
		t.Fatal("expected writer connection")
	}
	if database.ReadConn == nil {
		t.Fatal("expected reader connection")
	}
	if database.WriteQuerier == nil {
		t.Fatal("expected writer querier")
	}
	if database.ReadQuerier == nil {
		t.Fatal("expected reader querier")
	}
	if database.WriteConn == database.ReadConn {
		t.Fatal("expected separate writer and reader connection pools")
	}
	if got := database.WriteConn.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("expected one writer connection, got %d", got)
	}

	assertJournalMode(t, ctx, database.WriteConn, "writer")
	assertJournalMode(t, ctx, database.ReadConn, "reader")
	assertBusyTimeout(t, ctx, database.WriteConn, "writer")
	assertBusyTimeout(t, ctx, database.ReadConn, "reader")
	assertForeignKeys(t, ctx, database.WriteConn, "writer")
	assertForeignKeys(t, ctx, database.ReadConn, "reader")
}

func assertMigrationTableVersion(t *testing.T, ctx context.Context, database *Database, table string, expectedVersion int64) {
	t.Helper()

	var version int64
	var dirty bool
	if err := database.WriteConn.QueryRowContext(ctx, "SELECT version, dirty FROM "+table).Scan(&version, &dirty); err != nil {
		t.Fatalf("query %s: %v", table, err)
	}
	if version != expectedVersion {
		t.Fatalf("expected %s version %d, got %d", table, expectedVersion, version)
	}
	if dirty {
		t.Fatalf("expected %s to be clean", table)
	}
}

func assertJournalMode(t *testing.T, ctx context.Context, conn interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, name string) {
	t.Helper()

	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query %s journal mode: %v", name, err)
	}
	if mode != "wal" {
		t.Fatalf("expected %s journal mode wal, got %q", name, mode)
	}
}

func assertBusyTimeout(t *testing.T, ctx context.Context, conn interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, name string) {
	t.Helper()

	var timeoutMS int64
	if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeoutMS); err != nil {
		t.Fatalf("query %s busy timeout: %v", name, err)
	}
	if timeoutMS != 5000 {
		t.Fatalf("expected %s busy timeout 5000ms, got %dms", name, timeoutMS)
	}
}

func assertForeignKeys(t *testing.T, ctx context.Context, conn interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, name string) {
	t.Helper()

	var enabled int64
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("query %s foreign keys: %v", name, err)
	}
	if enabled != 1 {
		t.Fatalf("expected %s foreign keys on, got %d", name, enabled)
	}
}
