package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"slices"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"

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
	queries.Querier
	Conn *sql.DB
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
	if d == nil || d.Conn == nil {
		return nil
	}

	return d.Conn.Close()
}

func newSqlite(ctx context.Context, dataSourceName string) (*Database, error) {
	conn, err := sql.Open(string(EngineSqlite), dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	if err := runMigrations(ctx, conn, EngineSqlite); err != nil {
		conn.Close()
		return nil, err
	}

	return &Database{
		Querier: queries.New(conn),
		Conn:    conn,
	}, nil
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
