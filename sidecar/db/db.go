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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/mattn/go-sqlite3"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

//go:embed */schema/*.sql */seeds/*/*.sql
var migrationFiles embed.FS

//go:embed sqlite-vector/*/vector.*
var sqliteVectorExtensionFiles embed.FS

type Engine string

const (
	EngineSqlite Engine = "sqlite"

	sqliteVectorEntryPoint = "sqlite3_vector_init"
)

type Options struct {
	Engine                    Engine
	DataSourceName            string
	SQLiteVectorExtensionPath string
}

type Database struct {
	WriteQuerier              queries.Querier
	ReadQuerier               queries.Querier
	WriteConn                 *sql.DB
	ReadConn                  *sql.DB
	DataSourceName            string
	SQLiteVectorExtensionPath string
}

type databaseKey struct{}

func Register(registry *services.Services[any, any], database *Database) {
	services.Set(registry, databaseKey{}, database)
}

func FromServices(registry *services.Services[any, any]) (*Database, bool) {
	service, ok := services.Get[*Database](registry, databaseKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func New(ctx context.Context, opts Options) (*Database, error) {
	switch opts.Engine {
	case EngineSqlite:
		return newSqlite(ctx, opts.DataSourceName, opts.SQLiteVectorExtensionPath)
	default:
		return nil, fmt.Errorf("unsupported database engine %q", opts.Engine)
	}
}

func (d *Database) Close() error {
	if d == nil {
		return nil
	}

	var errs []error
	if d.WriteConn != nil {
		errs = append(errs, d.WriteConn.Close())
	}
	if d.ReadConn != nil {
		errs = append(errs, d.ReadConn.Close())
	}

	return errors.Join(errs...)
}

func newSqlite(ctx context.Context, dataSourceName string, sqliteVectorExtensionPath string) (*Database, error) {
	dataSourceName = SqliteDataSourceName(dataSourceName)
	sqliteVectorExtensionPath = ResolveSQLiteVectorExtensionPath(sqliteVectorExtensionPath)
	driverName := sqliteDriverName(sqliteVectorExtensionPath)

	writerConn, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open sqlite writer database: %w", err)
	}
	writerConn.SetMaxOpenConns(1)
	writerConn.SetMaxIdleConns(1)

	if err := writerConn.PingContext(ctx); err != nil {
		writerConn.Close()
		return nil, fmt.Errorf("ping sqlite writer database: %w", err)
	}

	if err := runMigrations(ctx, writerConn, EngineSqlite); err != nil {
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
		WriteQuerier:              queries.New(writerConn),
		ReadQuerier:               queries.New(readerConn),
		WriteConn:                 writerConn,
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

func runMigrations(ctx context.Context, conn *sql.DB, engine Engine) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := runMigrationDir(conn, path.Join(string(engine), "schema"), migratesqlite.DefaultMigrationsTable); err != nil {
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

func listSeedDirs(engine Engine) ([]string, error) {
	dir := path.Join(string(engine), "seeds")
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
