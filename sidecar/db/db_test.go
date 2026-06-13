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
