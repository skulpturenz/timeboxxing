package db

//go:generate go tool sqlc generate -f sqlc.yaml

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Options struct {
	DSN                       DSN
	SQLiteVectorExtensionPath *string
}

type Database struct {
	// WriteQuerier owns the write connection and the write lock (see SerialWriteQuerier); every write
	// — single statements, WriteTx transactions, and WithWriteConn maintenance — serializes through it.
	WriteQuerier *SerialWriteQuerier
	ReadQuerier  readqueries.Querier
	// ReadConn is the raw read connection, used by the semantic searcher / vector store for
	// sqlite-vector extension SQL that sqlc can't generate. Reads need no lock (WAL allows concurrent
	// readers alongside the single writer).
	ReadConn *sql.DB
	DSN      DSN
}

func Register(registry *services.Services[any, any], database *Database) {
	services.Set(registry, reflect.TypeFor[*Database](), database)
}

func FromServices(registry *services.Services[any, any]) (*Database, bool) {
	service, ok := services.Get[*Database](registry, reflect.TypeFor[*Database]())
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
