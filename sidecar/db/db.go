package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/mattn/go-sqlite3"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	enumsdbengine "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_db_engine"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

//go:embed schema/*.sql seeds/*/*.sql
var migrationFiles embed.FS

//go:embed sqlite-vector/*/vector.*
var sqliteVectorExtensionFiles embed.FS

const (
	sqliteVectorEntryPoint = "sqlite3_vector_init"
)

type Options struct {
	Engine                    enumsdbengine.DbEngine
	DataSourceName            string
	SQLiteVectorExtensionPath string
	// EncryptionKey, when non-empty, is the hex-encoded 32-byte (64 hex char) SQLCipher key used
	// to encrypt the database at rest. Empty means no encryption (dev / plaintext path).
	EncryptionKey string
}

// ErrDatabaseKeyMismatch is returned when the on-disk database is encrypted but the supplied key
// cannot decrypt it (e.g. a lost/rotated keychain entry). The database is left untouched so the
// caller can surface a non-destructive recovery message rather than reinitializing it.
var ErrDatabaseKeyMismatch = errors.New("database is encrypted but the provided key does not match")

type Database struct {
	// WriteQuerier owns the write connection and the write lock (see SerialWriteQuerier); every write
	// — single statements, WriteTx transactions, and WithWriteConn maintenance — serializes through it.
	WriteQuerier *SerialWriteQuerier
	ReadQuerier  readqueries.Querier
	// ReadConn is the raw read connection, used by the semantic searcher / vector store for
	// sqlite-vector extension SQL that sqlc can't generate. Reads need no lock (WAL allows concurrent
	// readers alongside the single writer).
	ReadConn                  *sql.DB
	DataSourceName            string
	SQLiteVectorExtensionPath string
}

func Register(registry *services.Services[any, any], database *Database) {
	services.Set(registry, reflect.TypeFor[Database](), database)
}

func FromServices(registry *services.Services[any, any]) (*Database, bool) {
	service, ok := services.Get[*Database](registry, reflect.TypeFor[Database]())
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func New(ctx context.Context, opts Options) (*Database, error) {
	switch opts.Engine {
	case enumsdbengine.Sqlite:
		return newSqlite(ctx, opts.DataSourceName, opts.SQLiteVectorExtensionPath, opts.EncryptionKey)
	default:
		return nil, fmt.Errorf("unsupported database engine %q", opts.Engine)
	}
}

func (d *Database) Close() error {
	if d == nil {
		return nil
	}

	var errs []error
	if d.WriteQuerier != nil && d.WriteQuerier.conn != nil {
		errs = append(errs, d.WriteQuerier.conn.Close())
	}
	if d.ReadConn != nil {
		errs = append(errs, d.ReadConn.Close())
	}

	return errors.Join(errs...)
}

func newSqlite(ctx context.Context, dataSourceName string, sqliteVectorExtensionPath string, encryptionKey string) (*Database, error) {
	encryptionKey = strings.TrimSpace(encryptionKey)
	if encryptionKey != "" && !isHexKey(encryptionKey) {
		return nil, fmt.Errorf("invalid database encryption key: expected hex-encoded bytes")
	}
	dbPath := sqliteFilePath(dataSourceName)

	// One-time, crash-safe migration of an existing plaintext database to an encrypted one. Runs
	// before any keyed connection is opened. No-op when no key is set or the file is already
	// encrypted / absent.
	if err := migrateToEncryptedIfNeeded(ctx, dbPath, encryptionKey); err != nil {
		return nil, fmt.Errorf("encrypt existing sqlite database: %w", err)
	}

	dataSourceName = SqliteKeyedDataSourceName(dataSourceName, encryptionKey)
	sqliteVectorExtensionPath = ResolveSQLiteVectorExtensionPath(sqliteVectorExtensionPath)
	driverName := sqliteDriverName(sqliteVectorExtensionPath)

	writerConn, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open sqlite writer database: %w", err)
	}
	// No connection-count restriction is needed: writeMu (via SerialWriteQuerier / WriteTx /
	// WithWriteLock) already guarantees a single writer at a time.
	writerConn.SetMaxIdleConns(1)

	if err := writerConn.PingContext(ctx); err != nil {
		writerConn.Close()
		if encryptionKey != "" && isEncryptedOnDisk(dbPath) {
			return nil, fmt.Errorf("%w: %v", ErrDatabaseKeyMismatch, err)
		}
		return nil, fmt.Errorf("ping sqlite writer database: %w", err)
	}

	// Force key validation with a real read before touching the schema, in case a connection was
	// established without reading a page. A mismatch here must NOT reinitialize the DB.
	if err := ensureDatabaseReadable(ctx, writerConn, dbPath, encryptionKey); err != nil {
		writerConn.Close()
		return nil, err
	}

	if err := runMigrations(ctx, writerConn, enumsdbengine.Sqlite); err != nil {
		writerConn.Close()
		return nil, err
	}

	// Fail closed: if a key was provided, the on-disk file must now be encrypted. This catches a
	// build that somehow lost the cipher and would otherwise silently persist plaintext.
	if err := verifyEncryptionAtRest(dbPath, encryptionKey); err != nil {
		writerConn.Close()
		return nil, err
	}

	readerConn, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		writerConn.Close()
		return nil, fmt.Errorf("open sqlite reader database: %w", err)
	}
	readerConn.SetMaxOpenConns(1)
	readerConn.SetMaxIdleConns(1)
	if err := readerConn.PingContext(ctx); err != nil {
		writerConn.Close()
		readerConn.Close()
		return nil, fmt.Errorf("ping sqlite reader database: %w", err)
	}

	return &Database{
		ReadQuerier:               readqueries.New(readerConn),
		WriteQuerier:              newSerialWriteQuerier(writerConn),
		ReadConn:                  readerConn,
		DataSourceName:            dataSourceName,
		SQLiteVectorExtensionPath: strings.TrimSpace(sqliteVectorExtensionPath),
	}, nil
}

func SqliteDataSourceName(dataSourceName string) string {
	base, rawQuery, hasQuery := strings.Cut(dataSourceName, "?")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		values = url.Values{}
	}
	values.Set("_journal_mode", "WAL")
	values.Set("_busy_timeout", "5000")
	values.Set("_foreign_keys", "on")

	if !hasQuery && values.Encode() == "" {
		return dataSourceName
	}
	return base + "?" + values.Encode()
}

// SqliteKeyedDataSourceName layers SQLCipher keying on top of SqliteDataSourceName. With an empty
// key it is identical to SqliteDataSourceName (the plaintext/dev path). With a key it selects the
// authenticated `sqlcipher` cipher scheme (AES-256-CBC + per-page HMAC) and supplies the raw
// 32-byte key as `x'<hex>'`. The single builder is reused for the writer, reader, AND the sqliteq
// queue connection so every connection to the file is keyed identically.
func SqliteKeyedDataSourceName(dataSourceName string, hexKey string) string {
	base := SqliteDataSourceName(dataSourceName)
	hexKey = strings.TrimSpace(hexKey)
	if hexKey == "" {
		return base
	}

	prefix, rawQuery, _ := strings.Cut(base, "?")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		values = url.Values{}
	}
	values.Set("_cipher", "sqlcipher")
	values.Set("_key", "x'"+hexKey+"'")
	return prefix + "?" + values.Encode()
}

func ResolveSQLiteVectorExtensionPath(configuredPath string) string {
	configuredPath = strings.TrimSpace(configuredPath)
	if configuredPath != "" {
		return configuredPath
	}

	for _, candidate := range sqliteVectorExtensionCandidates() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return extractBundledSQLiteVectorExtension()
}

func sqliteVectorExtensionCandidates() []string {
	resourcePath, ok := sqliteVectorResourcePath(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return nil
	}

	var baseDirs []string
	if executable, err := os.Executable(); err == nil {
		baseDirs = append(baseDirs, filepath.Dir(executable))
	}
	if workingDir, err := os.Getwd(); err == nil {
		for dir := workingDir; ; dir = filepath.Dir(dir) {
			baseDirs = append(baseDirs, dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	var candidates []string
	seen := map[string]struct{}{}
	addCandidate := func(candidate string) {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	for _, baseDir := range baseDirs {
		addCandidate(filepath.Join(baseDir, "sqlite-vector", resourcePath))
		addCandidate(filepath.Join(baseDir, "db", "sqlite-vector", resourcePath))
		addCandidate(filepath.Join(baseDir, "sidecar", "sqlite-vector", resourcePath))
		addCandidate(filepath.Join(baseDir, "sidecar", "db", "sqlite-vector", resourcePath))
	}
	return candidates
}

func sqliteVectorResourcePath(goos string, goarch string) (string, bool) {
	var suffix string
	switch goos {
	case "darwin":
		suffix = ".dylib"
	case "linux":
		suffix = ".so"
	case "windows":
		suffix = ".dll"
	default:
		return "", false
	}

	switch goarch {
	case "amd64", "arm64":
		return filepath.Join(goos+"-"+goarch, "vector"+suffix), true
	default:
		return "", false
	}
}

func extractBundledSQLiteVectorExtension() string {
	resourcePath, ok := sqliteVectorResourcePath(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return ""
	}

	data, err := sqliteVectorExtensionFiles.ReadFile(filepath.ToSlash(filepath.Join("sqlite-vector", resourcePath)))
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)
	targetDir := filepath.Join(
		os.TempDir(),
		"timeboxxing-sqlite-vector",
		hex.EncodeToString(sum[:8]),
		filepath.Dir(resourcePath),
	)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return ""
	}

	targetPath := filepath.Join(targetDir, filepath.Base(resourcePath))
	if existing, err := os.ReadFile(targetPath); err == nil && bytes.Equal(existing, data) {
		return targetPath
	}
	if err := os.WriteFile(targetPath, data, 0o755); err != nil {
		return ""
	}
	return targetPath
}

var (
	sqliteDriverMu    sync.Mutex
	sqliteDriverNames = map[string]string{}
)

func sqliteDriverName(sqliteVectorExtensionPath string) string {
	sqliteVectorExtensionPath = strings.TrimSpace(sqliteVectorExtensionPath)
	if sqliteVectorExtensionPath == "" {
		return "sqlite3"
	}

	sqliteDriverMu.Lock()
	defer sqliteDriverMu.Unlock()

	if driverName, ok := sqliteDriverNames[sqliteVectorExtensionPath]; ok {
		return driverName
	}

	sum := sha1.Sum([]byte(sqliteVectorExtensionPath))
	driverName := "timeboxxing_sqlite3_" + hex.EncodeToString(sum[:8])
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			if err := conn.LoadExtension(sqliteVectorExtensionPath, sqliteVectorEntryPoint); err != nil {
				return fmt.Errorf("load sqlite-vector extension %q: %w", sqliteVectorExtensionPath, err)
			}
			return nil
		},
	})
	sqliteDriverNames[sqliteVectorExtensionPath] = driverName
	return driverName
}

func runMigrations(ctx context.Context, conn *sql.DB, engine enumsdbengine.DbEngine) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := runMigrationDir(conn, path.Join(engine.String(), "schema"), migratesqlite.DefaultMigrationsTable); err != nil {
		return fmt.Errorf("run %s schema migrations: %w", engine, err)
	}

	seedDirs, err := listSeedDirs(engine)
	if err != nil {
		return fmt.Errorf("read %s seed migration directories: %w", engine, err)
	}

	for _, seedDir := range seedDirs {
		migrationTable := fmt.Sprintf("seed_migrations_%s", path.Base(seedDir))
		if err := runMigrationDir(conn, seedDir, migrationTable); err != nil {
			return fmt.Errorf("run %s seed migrations %s: %w", engine, seedDir, err)
		}
	}

	return nil
}

func runMigrationDir(conn *sql.DB, dir string, migrationsTable string) error {
	sourceDriver, err := iofs.New(migrationFiles, dir)
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseDriver, err := migratesqlite.WithInstance(conn, &migratesqlite.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance(dir, sourceDriver, "sqlite3", databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func listSeedDirs(engine enumsdbengine.DbEngine) ([]string, error) {
	dir := path.Join(engine.String(), "seeds")
	entries, err := migrationFiles.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirs = append(dirs, path.Join(dir, entry.Name()))
	}
	slices.Sort(dirs)

	return dirs, nil
}

const (
	sqliteHeaderMagic     = "SQLite format 3\x00"
	migrationTempSuffix   = ".encrypting.tmp"
	migrationBackupSuffix = ".plaintext.bak"
)

// sqliteFilePath extracts the on-disk file path from a sqlite DSN (strips the query string and an
// optional file: scheme prefix). Mirrors the DSN handling in grpc/settings/database_maintenance.go.
func sqliteFilePath(dataSourceName string) string {
	filePath := dataSourceName
	if i := strings.IndexByte(filePath, '?'); i >= 0 {
		filePath = filePath[:i]
	}
	return strings.TrimPrefix(filePath, "file:")
}

func isHexKey(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// isPlaintextSQLite reports whether the file at path begins with the well-known unencrypted SQLite
// header. An encrypted SQLCipher database starts with a random salt instead. A file too small to
// contain the header (e.g. a freshly created 0-byte file) is treated as not-plaintext.
func isPlaintextSQLite(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	header := make([]byte, len(sqliteHeaderMagic))
	if _, err := io.ReadFull(f, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	return string(header) == sqliteHeaderMagic, nil
}

// migrationKeyedDSN builds a keyed DSN for the transient connections used during migration and
// verification. It deliberately omits WAL so these short-lived opens do not leave -wal/-shm files
// next to the temp file.
func migrationKeyedDSN(path string, hexKey string) string {
	return fmt.Sprintf("%s?_cipher=sqlcipher&_key=x'%s'&_busy_timeout=5000", path, hexKey)
}

// migrateToEncryptedIfNeeded converts an existing plaintext database into an encrypted one exactly
// once, using a copy-then-rekey strategy that never mutates the original until the ciphertext is
// verified. It is a no-op when no key is set, the file is absent (new install), or the file is
// already encrypted. It is idempotent and crash-safe.
func migrateToEncryptedIfNeeded(ctx context.Context, dbPath string, hexKey string) error {
	if hexKey == "" || dbPath == "" || dbPath == ":memory:" {
		return nil
	}

	if err := recoverInterruptedMigration(ctx, dbPath, hexKey); err != nil {
		return fmt.Errorf("recover interrupted migration: %w", err)
	}

	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil // new install: the keyed connection creates an encrypted DB on first write
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("database path %q is a directory", dbPath)
	}

	plaintext, err := isPlaintextSQLite(dbPath)
	if err != nil {
		return err
	}
	if !plaintext {
		return nil // already encrypted (or not a plaintext SQLite file)
	}

	tmpPath := dbPath + migrationTempSuffix
	bakPath := dbPath + migrationBackupSuffix
	_ = os.Remove(tmpPath)

	// Fold any WAL frames into the main file so the plain copy is complete.
	if err := checkpointPlaintextWAL(ctx, dbPath); err != nil {
		return fmt.Errorf("checkpoint before encrypting: %w", err)
	}
	if err := copyFile(dbPath, tmpPath); err != nil {
		return fmt.Errorf("copy database before encrypting: %w", err)
	}
	if err := rekeyPlaintextCopy(ctx, tmpPath, hexKey); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encrypt database copy: %w", err)
	}
	if err := verifyEncryptedCopy(ctx, tmpPath, hexKey); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("verify encrypted database: %w", err)
	}

	// Swap only after the ciphertext is proven good. Rename-only; original preserved as backup.
	_ = os.Remove(bakPath)
	if err := os.Rename(dbPath, bakPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("back up plaintext database: %w", err)
	}
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = os.Rename(bakPath, dbPath) // best-effort restore of the original
		return fmt.Errorf("promote encrypted database: %w", err)
	}
	// Stale plaintext WAL/SHM belong to the old file and are invalid for the encrypted one.
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	// At-rest encryption must not leave a plaintext copy behind.
	_ = os.Remove(bakPath)
	return nil
}

// recoverInterruptedMigration resolves the state left by a crash during a prior migration so the
// next launch converges on either the intact plaintext DB (to be re-migrated) or the verified
// encrypted DB, never a half-written one.
func recoverInterruptedMigration(ctx context.Context, dbPath string, hexKey string) error {
	tmpPath := dbPath + migrationTempSuffix
	bakPath := dbPath + migrationBackupSuffix

	_, mainErr := os.Stat(dbPath)
	mainMissing := errors.Is(mainErr, os.ErrNotExist)
	_, tmpErr := os.Stat(tmpPath)
	tmpExists := tmpErr == nil

	switch {
	case mainMissing && tmpExists:
		// Crashed between the two renames. Promote the tmp if it decrypts, else roll back.
		if databaseOpensWithKey(ctx, tmpPath, hexKey) {
			if err := os.Rename(tmpPath, dbPath); err != nil {
				return err
			}
			_ = os.Remove(bakPath)
		} else if _, err := os.Stat(bakPath); err == nil {
			if err := os.Rename(bakPath, dbPath); err != nil {
				return err
			}
			_ = os.Remove(tmpPath)
		}
	case !mainMissing:
		// The original is intact; any tmp is a stale partial from an aborted run.
		_ = os.Remove(tmpPath)
	}

	// Clean a leftover plaintext backup once the main DB is present again.
	if _, err := os.Stat(dbPath); err == nil {
		if _, err := os.Stat(bakPath); err == nil {
			_ = os.Remove(bakPath)
		}
	}
	return nil
}

func checkpointPlaintextWAL(ctx context.Context, dbPath string) error {
	conn, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if _, err := conn.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "PRAGMA journal_mode=DELETE")
	return err
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// rekeyPlaintextCopy opens the plaintext copy with the target cipher configured but no key (so it
// opens as plaintext) and encrypts it in place via PRAGMA rekey.
func rekeyPlaintextCopy(ctx context.Context, path string, hexKey string) error {
	conn, err := sql.Open("sqlite3", path+"?_cipher=sqlcipher")
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	_, err = conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA rekey = "x'%s'";`, hexKey))
	return err
}

// verifyEncryptedCopy confirms the freshly encrypted file is no longer plaintext, opens with the
// key, and passes an integrity check — all before the original is touched.
func verifyEncryptedCopy(ctx context.Context, path string, hexKey string) error {
	plaintext, err := isPlaintextSQLite(path)
	if err != nil {
		return err
	}
	if plaintext {
		return fmt.Errorf("encrypted copy is still plaintext")
	}

	conn, err := sql.Open("sqlite3", migrationKeyedDSN(path, hexKey))
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	var result string
	if err := conn.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}
	return nil
}

func databaseOpensWithKey(ctx context.Context, path string, hexKey string) bool {
	conn, err := sql.Open("sqlite3", migrationKeyedDSN(path, hexKey))
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	var count int
	return conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&count) == nil
}

// ensureDatabaseReadable forces SQLCipher key validation with a real read. SQLCipher only rejects a
// wrong key on first page access, so a bad key surfaces here rather than at connect. When the key is
// wrong for an already-encrypted file it returns ErrDatabaseKeyMismatch so the caller can report a
// non-destructive recovery error instead of reinitializing the database.
func ensureDatabaseReadable(ctx context.Context, conn *sql.DB, dbPath string, hexKey string) error {
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		if hexKey != "" && isEncryptedOnDisk(dbPath) {
			return fmt.Errorf("%w: %v", ErrDatabaseKeyMismatch, err)
		}
		return fmt.Errorf("read sqlite database: %w", err)
	}
	return nil
}

// isEncryptedOnDisk reports whether the file at dbPath exists and does not carry the plaintext
// SQLite header (i.e. it looks like an encrypted database).
func isEncryptedOnDisk(dbPath string) bool {
	plaintext, err := isPlaintextSQLite(dbPath)
	if err != nil {
		return false
	}
	return !plaintext
}

// verifyEncryptionAtRest fails closed if a key was provided but the database on disk is not
// encrypted — guarding against a build that lost the cipher and would silently persist plaintext.
func verifyEncryptionAtRest(dbPath string, hexKey string) error {
	if hexKey == "" || dbPath == "" || dbPath == ":memory:" {
		return nil
	}
	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) || (err == nil && info.Size() == 0) {
		return nil // nothing persisted yet
	}
	if err != nil {
		return err
	}
	plaintext, err := isPlaintextSQLite(dbPath)
	if err != nil {
		return err
	}
	if plaintext {
		return fmt.Errorf("database key was provided but %q is not encrypted on disk", dbPath)
	}
	return nil
}
