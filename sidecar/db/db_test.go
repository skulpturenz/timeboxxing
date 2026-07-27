package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	sqlitevector "github.com/skulpturenz/timeboxxing/sidecar/db/sqlite-vector"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
)

func ptr[T any](v T) *T { return &v }

func TestSQLiteVectorExtensionIsLoadedWhenBundledOrConfigured(t *testing.T) {
	ctx := context.Background()
	configuredPath := os.Getenv("SIDECAR_SQLITE_VECTOR_EXTENSION_PATH")
	if !sqliteVectorExtensionAvailable(configuredPath) {
		t.Skip("sqlite-vector extension is not bundled for this platform and SIDECAR_SQLITE_VECTOR_EXTENSION_PATH is not set")
	}
	var configuredPathOption *string
	if strings.TrimSpace(configuredPath) != "" {
		configuredPathOption = &configuredPath
	}
	database, err := New(ctx, Options{
		DSN:                       NewDSN(filepath.Join(t.TempDir(), "test.db")),
		SQLiteVectorExtensionPath: configuredPathOption,
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	var version string
	if err := database.WriteQuerier.conn.QueryRowContext(ctx, `SELECT vector_version()`).Scan(&version); err != nil {
		t.Fatalf("query sqlite-vector version: %v", err)
	}
	if version == "" {
		t.Fatal("expected sqlite-vector version")
	}
}

func TestSQLiteVectorExtensionIsLoadedFromEmbeddedBundle(t *testing.T) {
	ctx := context.Background()
	if !sqliteVectorExtensionAvailable("") {
		t.Skip("sqlite-vector extension is not embedded for this platform")
	}
	// No configured path — the bundled extension is loaded automatically.
	database, err := New(ctx, Options{
		DSN: NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	var version string
	if err := database.WriteQuerier.conn.QueryRowContext(ctx, `SELECT vector_version()`).Scan(&version); err != nil {
		t.Fatalf("query sqlite-vector version: %v", err)
	}
	if version == "" {
		t.Fatal("expected sqlite-vector version")
	}
}

func TestSqliteMigrationsRunOnce(t *testing.T) {
	ctx := context.Background()
	dsn := NewDSN(filepath.Join(t.TempDir(), "test.db"))

	database, err := New(ctx, Options{DSN: dsn})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	database, err = New(ctx, Options{DSN: dsn})
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	defer database.Close()

	assertMigrationTableVersion(t, ctx, database, "schema_migrations", 22)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_models", 1)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_model_providers", 1)
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_application_settings", 1)
}

func TestSqliteProjectsMigrationCreatesTable(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		DSN: NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	if _, err := database.WriteQuerier.CreateProject(ctx, "Client Work"); err != nil {
		t.Fatalf("create project: %v", err)
	}
	projects, err := database.ReadQuerier.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "Client Work" {
		t.Fatalf("expected project name Client Work, got %q", projects[0].Name)
	}
	// Colour/rate live in project_details now; a project created without details has neither.
	if projects[0].ColorArgb != nil {
		t.Fatalf("expected no colour, got %d", *projects[0].ColorArgb)
	}
	if projects[0].HourlyRateCents != 0 {
		t.Fatalf("expected zero hourly rate, got %d", projects[0].HourlyRateCents)
	}
}

func TestSqliteTimesheetsMigrationCreatesTables(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		DSN: NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	dayStart := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	startedAt := dayStart.Add(9 * time.Hour)
	// The ledger is not seeded; it is created lazily before the first ledger item.
	if err := database.WriteQuerier.EnsureLedger(ctx); err != nil {
		t.Fatalf("ensure ledger: %v", err)
	}
	entry, err := database.WriteQuerier.CreateLedgerItem(ctx, writequeries.CreateLedgerItemParams{
		Billable:     true,
		Title:        "Design review",
		Notes:        nil,
		StartedAtUtc: ptr(startedAt),
		EndedAtUtc:   ptr(startedAt.Add(30 * time.Minute)),
	})
	if err != nil {
		t.Fatalf("create ledger item: %v", err)
	}
	// A ledger item links to timeline entries (usage blocks) via ledger_item_timeline_entries.
	if err := database.WriteQuerier.CreateLedgerItemTimelineEntry(ctx, writequeries.CreateLedgerItemTimelineEntryParams{
		LedgerItemsID: ptr(entry.ID),
		TimelineID:    nil,
	}); err != nil {
		t.Fatalf("create ledger item timeline entry: %v", err)
	}

	entries, err := database.ReadQuerier.ListTimesheetEntries(ctx, readqueries.ListTimesheetEntriesParams{
		StartedAtUtc:   ptr(dayStart),
		StartedAtUtc_2: ptr(dayStart.Add(24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("list timesheet entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 timesheet entry, got %d", len(entries))
	}
}

func TestSqliteUsesWALAndSeparatePools(t *testing.T) {
	ctx := context.Background()
	dsn := NewDSN(filepath.Join(t.TempDir(), "test.db"))
	dsn.SetJournalMode(enumsjournalmode.WAL)
	dsn.EnableFK()
	dsn.SetBusyTimeout(5 * time.Second)
	database, err := New(ctx, Options{DSN: dsn})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	if database.WriteQuerier.conn == nil {
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
	if database.WriteQuerier.conn == database.ReadConn {
		t.Fatal("expected separate writer and reader connection pools")
	}

	assertJournalMode(t, ctx, database.WriteQuerier.conn, "writer")
	assertJournalMode(t, ctx, database.ReadConn, "reader")
	assertBusyTimeout(t, ctx, database.WriteQuerier.conn, "writer")
	assertBusyTimeout(t, ctx, database.ReadConn, "reader")
	assertForeignKeys(t, ctx, database.WriteQuerier.conn, "writer")
	assertForeignKeys(t, ctx, database.ReadConn, "reader")
}

func TestSqliteSerializesConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	database, err := New(ctx, Options{
		DSN: NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer database.Close()

	// With the writer connection pool no longer capped at 1, the write mutex is what keeps SQLite
	// from ever seeing two concurrent writers. Hammer both write paths (the serial querier and
	// WriteTx) from many goroutines and assert none of them observe SQLITE_BUSY or any other error.
	const writers = 16
	const perWriter = 25
	var wg sync.WaitGroup
	errs := make(chan error, writers*perWriter)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if i%2 == 0 {
					if _, err := database.WriteQuerier.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
						Name: fmt.Sprintf("app-%d-%d", w, i),
					}); err != nil {
						errs <- err
					}
					continue
				}
				if err := database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
					_, err := q.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
						Name: fmt.Sprintf("app-tx-%d-%d", w, i),
					})
					return err
				}); err != nil {
					errs <- err
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent write failed: %v", err)
	}
}

// sqliteVectorExtensionAvailable reports whether the sqlite-vector extension can be loaded — from the
// configured path when set, otherwise from the bundle embedded in the binary. Tests skip when it is
// unavailable for the current platform.
func sqliteVectorExtensionAvailable(configuredPath string) bool {
	options := sqlitevector.Options{}
	if strings.TrimSpace(configuredPath) != "" {
		options.Path = &configuredPath
	}
	_, _, err := options.Load()
	return err == nil
}

func assertMigrationTableVersion(t *testing.T, ctx context.Context, database *Database, table string, expectedVersion int64) {
	t.Helper()

	var version int64
	var dirty bool
	if err := database.WriteQuerier.conn.QueryRowContext(ctx, "SELECT version, dirty FROM "+table).Scan(&version, &dirty); err != nil {
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
