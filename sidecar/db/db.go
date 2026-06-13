package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"

	_ "modernc.org/sqlite"
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
	if err := runSchemaMigrations(ctx, conn, engine); err != nil {
		return err
	}

	return runSeedScripts(ctx, conn, engine)
}

func runSchemaMigrations(ctx context.Context, conn *sql.DB, engine Engine) error {
	migrations, err := listFlatSQLFiles(path.Join(string(engine), "schema"))
	if err != nil {
		return fmt.Errorf("read %s migrations: %w", engine, err)
	}

	for _, migration := range migrations {
		if err := runSQLFile(ctx, conn, migration); err != nil {
			return fmt.Errorf("run %s migration %s: %w", engine, migration, err)
		}
	}

	return nil
}

func runSeedScripts(ctx context.Context, conn *sql.DB, engine Engine) error {
	seeds, err := listRecursiveSQLFiles(path.Join(string(engine), "seeds"))
	if err != nil {
		return fmt.Errorf("read %s seed scripts: %w", engine, err)
	}

	for _, seed := range seeds {
		if err := runSQLFile(ctx, conn, seed); err != nil {
			return fmt.Errorf("run %s seed script %s: %w", engine, seed, err)
		}
	}

	return nil
}

func listFlatSQLFiles(dir string) ([]string, error) {
	entries, err := migrationFiles.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, path.Join(dir, entry.Name()))
	}
	slices.Sort(files)

	return files, nil
}

func listRecursiveSQLFiles(dir string) ([]string, error) {
	files := []string{}
	err := fs.WalkDir(migrationFiles, dir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			return nil
		}
		files = append(files, filePath)
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	slices.Sort(files)

	return files, nil
}

func runSQLFile(ctx context.Context, conn *sql.DB, filePath string) error {
	contents, err := migrationFiles.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read SQL file: %w", err)
	}

	_, err = conn.ExecContext(ctx, string(contents))
	return err
}
