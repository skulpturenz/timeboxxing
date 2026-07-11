package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"reflect"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	sqlitevector "github.com/skulpturenz/timeboxxing/sidecar/db/sqlite-vector"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

//go:embed sqlite-vector/*/vector.*
var sqliteVectorExtensionFiles embed.FS

const (
	driverName = "timeboxxing_sqlite"
)

type Options struct {
	DSN                       DSN
	SQLiteVectorExtensionPath string
}

type Database struct {
	// WriteQuerier owns the write connection and the write lock (see SerialWriteQuerier); every write
	// — single statements, WriteTx transactions, and WithWriteConn maintenance — serializes through it.
	WriteQuerier *SerialWriteQuerier
	ReadQuerier  readqueries.Querier
	// ReadConn is the raw read connection, used by the semantic searcher / vector store for
	// sqlite-vector extension SQL that sqlc can't generate. Reads need no lock (WAL allows concurrent
	// readers alongside the single writer).
	ReadConn                  *sql.DB
	DSN                       DSN
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
	return newSqlite(ctx, opts.DSN, opts.SQLiteVectorExtensionPath)
}

func (d *Database) Close() error {
	var errs []error
	if d.WriteQuerier != nil && d.WriteQuerier.conn != nil {
		errs = append(errs, d.WriteQuerier.conn.Close())
	}
	if d.ReadConn != nil {
		errs = append(errs, d.ReadConn.Close())
	}

	return errors.Join(errs...)
}

func newSqlite(ctx context.Context, dsn DSN, sqliteVectorExtensionPath string) (*Database, error) {
	sqliteVectorOptions := sqlitevector.Options{
		Path: &sqliteVectorExtensionPath,
	}
	registerExtensions(driverName, sqliteVectorOptions.Load)

	writerConn, err := sql.Open(driverName, dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open sqlite writer database: %w", err)
	}

	if err := writerConn.PingContext(ctx); err != nil {
		writerConn.Close()

		return nil, fmt.Errorf("ping sqlite writer database: %w", err)
	}

	if err := ensureDatabaseReadable(ctx, writerConn); err != nil {
		writerConn.Close()
		return nil, err
	}

	if err := runSchemaMigrations(ctx, writerConn); err != nil {
		writerConn.Close()
		return nil, err
	}

	if err := runSeedMigrations(ctx, writerConn); err != nil {
		writerConn.Close()
		return nil, err
	}

	readerConn, err := sql.Open(driverName, dsn.String())
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
		ReadQuerier:  readqueries.New(readerConn),
		WriteQuerier: newSerialWriteQuerier(writerConn),
		ReadConn:     readerConn,
		DSN:          dsn,
	}, nil
}

func ensureDatabaseReadable(ctx context.Context, conn *sql.DB) error {
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		return fmt.Errorf("read sqlite database: %w", err)
	}

	return nil
}
