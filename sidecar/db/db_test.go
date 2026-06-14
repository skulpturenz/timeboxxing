package db

import (
	"context"
	"path/filepath"
	"testing"
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
	if err := database.Conn.QueryRowContext(ctx, `SELECT vec_version()`).Scan(&version); err != nil {
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
	assertMigrationTableVersion(t, ctx, database, "seed_migrations_transition_event_reasons", 1)
}

func assertMigrationTableVersion(t *testing.T, ctx context.Context, database *Database, table string, expectedVersion int64) {
	t.Helper()

	var version int64
	var dirty bool
	if err := database.Conn.QueryRowContext(ctx, "SELECT version, dirty FROM "+table).Scan(&version, &dirty); err != nil {
		t.Fatalf("query %s: %v", table, err)
	}
	if version != expectedVersion {
		t.Fatalf("expected %s version %d, got %d", table, expectedVersion, version)
	}
	if dirty {
		t.Fatalf("expected %s to be clean", table)
	}
}
