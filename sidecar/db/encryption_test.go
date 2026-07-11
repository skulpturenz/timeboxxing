package db

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	enumsdbengine "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_db_engine"
)

const (
	testKey      = "2dd29ca851e7b56e4697b0e1f08507293d761a05ce4d1b628663f411a8086d99"
	testWrongKey = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// A recognizable string persisted into the DB; it must never appear in the encrypted file.
	projectMarker = "PLAINTEXT-MARKER-Client-Work-Alpha"
)

func fileHeader(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	buf := make([]byte, len(sqliteHeaderMagic))
	_, _ = f.Read(buf)
	return buf
}

func fileHasMarker(t *testing.T, path, marker string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Contains(string(data), marker)
}

func insertMarkerProject(t *testing.T, ctx context.Context, d *Database) {
	t.Helper()
	if _, err := d.WriteQuerier.CreateProject(ctx, writequeries.CreateProjectParams{
		ID:        "marker",
		Name:      projectMarker,
		ColorArgb: 0xFF00FFEE,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create marker project: %v", err)
	}
}

func TestSqliteKeyedDataSourceName(t *testing.T) {
	// No key -> identical to the plaintext builder.
	if got, want := SqliteKeyedDataSourceName("test.db", ""), SqliteDataSourceName("test.db"); got != want {
		t.Fatalf("empty key: got %q want %q", got, want)
	}
	// With key -> selects the sqlcipher scheme and carries the raw key.
	dsn := SqliteKeyedDataSourceName("test.db", testKey)
	if !strings.Contains(dsn, "_cipher=sqlcipher") {
		t.Fatalf("keyed DSN missing _cipher: %q", dsn)
	}
	if !strings.Contains(dsn, "_key=") {
		t.Fatalf("keyed DSN missing _key: %q", dsn)
	}
	// It must still carry the base pragmas.
	for _, want := range []string{"_journal_mode=WAL", "_busy_timeout=5000", "_foreign_keys=on"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("keyed DSN missing %q: %q", want, dsn)
		}
	}
}

func TestEncryptedDatabaseOnDiskAndKeyRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "enc.db")

	d, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testKey})
	if err != nil {
		t.Fatalf("create encrypted database: %v", err)
	}
	insertMarkerProject(t, ctx, d)
	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// On-disk: encrypted header, and the plaintext marker must be absent.
	if got := string(fileHeader(t, dbPath)); got == sqliteHeaderMagic {
		t.Fatalf("database is plaintext on disk")
	}
	if fileHasMarker(t, dbPath, projectMarker) {
		t.Fatalf("plaintext marker leaked into encrypted file")
	}

	// The DataSourceName the queue reuses must be keyed.
	if !strings.Contains(d.DataSourceName, "_cipher=sqlcipher") {
		t.Fatalf("Database.DataSourceName is not keyed: %q", d.DataSourceName)
	}

	// Reopen with the correct key -> data intact.
	d2, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testKey})
	if err != nil {
		t.Fatalf("reopen with key: %v", err)
	}
	defer d2.Close()
	projects, err := d2.ReadQuerier.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != projectMarker {
		t.Fatalf("marker project not readable after reopen: %+v", projects)
	}
}

func TestWrongKeyFailsSafely(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "enc.db")

	d, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testKey})
	if err != nil {
		t.Fatalf("create encrypted database: %v", err)
	}
	insertMarkerProject(t, ctx, d)
	d.Close()
	before := fileHeader(t, dbPath)

	// Wrong key must return ErrDatabaseKeyMismatch and NOT modify the file.
	_, err = New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testWrongKey})
	if !errors.Is(err, ErrDatabaseKeyMismatch) {
		t.Fatalf("expected ErrDatabaseKeyMismatch, got %v", err)
	}
	if got := fileHeader(t, dbPath); string(got) != string(before) {
		t.Fatalf("database header changed after a wrong-key open attempt")
	}

	// Missing key (empty) on an encrypted DB must also fail without destroying it.
	_, err = New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath})
	if err == nil {
		t.Fatalf("expected error opening encrypted DB with no key")
	}
	if got := fileHeader(t, dbPath); string(got) != string(before) {
		t.Fatalf("database header changed after a no-key open attempt")
	}
}

func TestPlaintextDatabaseMigratesToEncrypted(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	// 1) Build a legacy PLAINTEXT database (no key) with recognizable data.
	legacy, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath})
	if err != nil {
		t.Fatalf("create plaintext database: %v", err)
	}
	insertMarkerProject(t, ctx, legacy)
	legacy.Close()

	if got := string(fileHeader(t, dbPath)); got != sqliteHeaderMagic {
		t.Fatalf("precondition: legacy DB is not plaintext on disk")
	}
	if !fileHasMarker(t, dbPath, projectMarker) {
		t.Fatalf("precondition: marker not present in plaintext DB")
	}

	// 2) Open WITH a key -> triggers the one-time migration.
	migrated, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testKey})
	if err != nil {
		t.Fatalf("open triggers migration: %v", err)
	}

	// Encrypted on disk, data intact, no plaintext leak, temp/backup files cleaned up.
	if got := string(fileHeader(t, dbPath)); got == sqliteHeaderMagic {
		t.Fatalf("database still plaintext after migration")
	}
	if fileHasMarker(t, dbPath, projectMarker) {
		t.Fatalf("plaintext marker leaked after migration")
	}
	projects, err := migrated.ReadQuerier.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list after migration: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != projectMarker {
		t.Fatalf("data lost in migration: %+v", projects)
	}
	migrated.Close()

	for _, leftover := range []string{dbPath + migrationTempSuffix, dbPath + migrationBackupSuffix} {
		if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("leftover file not cleaned up: %s", leftover)
		}
	}

	// 3) Idempotent: a second keyed open performs no migration and still reads the data.
	again, err := New(ctx, Options{Engine: enumsdbengine.Sqlite, DataSourceName: dbPath, EncryptionKey: testKey})
	if err != nil {
		t.Fatalf("reopen migrated database: %v", err)
	}
	defer again.Close()
	projects, err = again.ReadQuerier.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list after reopen: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != projectMarker {
		t.Fatalf("data changed on idempotent reopen: %+v", projects)
	}
}
