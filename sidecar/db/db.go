package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	_ "modernc.org/sqlite"
)

//go:embed */schema/*.sql
var migrationFiles embed.FS

type Engine string

const (
	EngineSqlite Engine = "sqlite"
)

type Options struct {
	Engine         Engine
	DataSourceName string
}

type Querier = queries.Querier

type Database struct {
	Conn    *sql.DB
	Queries Querier
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
		Conn:    conn,
		Queries: queries.New(conn),
	}, nil
}

func runMigrations(ctx context.Context, conn *sql.DB, engine Engine) error {
	dir := path.Join(string(engine), "schema")
	entries, err := migrationFiles.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s migrations: %w", engine, err)
	}

	migrations := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		migrations = append(migrations, path.Join(dir, entry.Name()))
	}
	slices.Sort(migrations)

	for _, migration := range migrations {
		contents, err := migrationFiles.ReadFile(migration)
		if err != nil {
			return fmt.Errorf("read %s migration %s: %w", engine, migration, err)
		}

		if _, err := conn.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("run %s migration %s: %w", engine, migration, err)
		}
	}

	return nil
}
