package db

import (
	"context"
	"database/sql"
	"fmt"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	sqlitevector "github.com/skulpturenz/timeboxxing/sidecar/db/sqlite-vector"
)

const driverName = "timeboxxing_sqlite"

func newSqlite(ctx context.Context, dsn DSN, sqliteVectorExtensionPath *string) (*Database, error) {
	sqliteVectorOptions := sqlitevector.Options{
		Path: sqliteVectorExtensionPath,
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

	// if the encryption key is wrong, we only know about it when we try to query
	var count int
	if err := writerConn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		return nil, fmt.Errorf("read sqlite database: %w", err)
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
