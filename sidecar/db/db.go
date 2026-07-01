package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

//go:embed */schema/*.sql */seeds/*/*.sql
var migrationFiles embed.FS

type Engine string

const (
	EngineSqlite Engine = "sqlite"
)

type Options struct {
	Engine         Engine
	DataSourceName string
}

type Database struct {
	WriteQuerier   queries.Querier
	ReadQuerier    queries.Querier
	WriteConn      *sql.DB
	ReadConn       *sql.DB
	DataSourceName string
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
		return newSqlite(ctx, opts.DataSourceName)
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

func newSqlite(ctx context.Context, dataSourceName string) (*Database, error) {
	dataSourceName = SqliteDataSourceName(dataSourceName)

	writerConn, err := sql.Open(string(EngineSqlite), dataSourceName)
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

	readerConn, err := sql.Open(string(EngineSqlite), dataSourceName)
	if err != nil {
		writerConn.Close()
		return nil, fmt.Errorf("open sqlite reader database: %w", err)
	}
	if err := readerConn.PingContext(ctx); err != nil {
		writerConn.Close()
		readerConn.Close()
		return nil, fmt.Errorf("ping sqlite reader database: %w", err)
	}

	return &Database{
		WriteQuerier:   queries.New(writerConn),
		ReadQuerier:    queries.New(readerConn),
		WriteConn:      writerConn,
		ReadConn:       readerConn,
		DataSourceName: dataSourceName,
	}, nil
}

func SqliteDataSourceName(dataSourceName string) string {
	base, rawQuery, hasQuery := strings.Cut(dataSourceName, "?")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		values = url.Values{}
	}
	values.Set("_journal_mode", "WAL")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "busy_timeout(5000)")
	values.Add("_pragma", "foreign_keys(ON)")

	if !hasQuery && values.Encode() == "" {
		return dataSourceName
	}
	return base + "?" + values.Encode()
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

	migrator, err := migrate.NewWithInstance(dir, sourceDriver, string(EngineSqlite), databaseDriver)
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
