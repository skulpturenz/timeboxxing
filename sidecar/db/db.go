package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	_ "modernc.org/sqlite"
)

//go:embed sql/schema.sql
var schemaFiles embed.FS

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

	if err := initializeSchema(ctx, conn); err != nil {
		conn.Close()
		return nil, err
	}

	return &Database{
		Conn:    conn,
		Queries: queries.New(conn),
	}, nil
}

func initializeSchema(ctx context.Context, conn *sql.DB) error {
	schema, err := schemaFiles.ReadFile("sql/schema.sql")
	if err != nil {
		return fmt.Errorf("read database schema: %w", err)
	}

	if _, err := conn.ExecContext(ctx, string(schema)); err != nil {
		return fmt.Errorf("initialize database schema: %w", err)
	}

	return nil
}
