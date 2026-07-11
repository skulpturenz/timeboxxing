package settings

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetDatabaseMaintenanceStatus(context.Context, *settingsv1.GetDatabaseMaintenanceStatusRequest) (*settingsv1.DatabaseMaintenanceStatus, error) {
	if s.database == nil {
		return nil, status.Error(codes.FailedPrecondition, "database is unavailable")
	}

	size, err := sqliteFootprintSize(s.database.DSN.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size: %v", err)
	}
	return &settingsv1.DatabaseMaintenanceStatus{SizeBytes: size}, nil
}

func (s *Server) PruneDatabaseRange(ctx context.Context, req *settingsv1.PruneDatabaseRangeRequest) (*settingsv1.PruneDatabaseRangeResponse, error) {
	if s.database == nil || s.database.WriteQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "database is unavailable")
	}

	startedAt, endedAt, err := pruneWindowFromRequest(req)
	if err != nil {
		return nil, err
	}

	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	// Hold the shared write lock for the whole prune transaction so it serializes against every
	// other writer (the usage monitor, gRPC writes) rather than racing them on the write connection.
	var counts pruneCounts
	if err := s.database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return status.Errorf(codes.Internal, "begin prune transaction: %v", err)
		}
		defer tx.Rollback()

		cleanupNeeded := true
		defer func() {
			if cleanupNeeded {
				_ = dropPruneTempTables(ctx, tx)
			}
		}()

		if err := createPruneTempTables(ctx, tx, startedAt, endedAt); err != nil {
			return status.Errorf(codes.Internal, "prepare prune scope: %v", err)
		}

		counts, err = deletePruneRows(ctx, tx)
		if err != nil {
			return status.Errorf(codes.Internal, "delete pruned rows: %v", err)
		}

		if err := dropPruneTempTables(ctx, tx); err != nil {
			return status.Errorf(codes.Internal, "clean prune scope: %v", err)
		}
		cleanupNeeded = false

		if err := tx.Commit(); err != nil {
			return status.Errorf(codes.Internal, "commit prune transaction: %v", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	size, err := sqliteFootprintSize(s.database.DSN.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload database size: %v", err)
	}

	return &settingsv1.PruneDatabaseRangeResponse{
		SizeBytes:                        size,
		LedgerItemsDeleted:               counts.ledgerItemsDeleted,
		LedgerItemTimelineEntriesDeleted: counts.ledgerItemTimelineEntriesDeleted,
		TimelineDeleted:                  counts.timelineDeleted,
		ForegroundProcessesDeleted:       counts.foregroundProcessesDeleted,
		ForegroundProcessMetadataDeleted: counts.foregroundProcessMetadataDeleted,
		TimelineSemanticDocumentsDeleted: counts.timelineSemanticDocumentsDeleted,
		TimelineEmbeddingsDeleted:        counts.timelineEmbeddingsDeleted,
		ApplicationsDeleted:              counts.applicationsDeleted,
	}, nil
}

func (s *Server) VacuumDatabase(ctx context.Context, _ *settingsv1.VacuumDatabaseRequest) (*settingsv1.VacuumDatabaseResponse, error) {
	if s.database == nil || s.database.WriteQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "database is unavailable")
	}

	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	sizeBefore, err := sqliteFootprintSize(s.database.DSN.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size before vacuum: %v", err)
	}

	// VACUUM cannot run inside a transaction, so serialize it against other writers with the write
	// lock directly rather than via WriteTx.
	if err := s.database.WriteQuerier.WithWriteConn(func(conn *sql.DB) error {
		return vacuumSQLiteDatabase(ctx, conn)
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "vacuum database: %v", err)
	}

	sizeAfter, err := sqliteFootprintSize(s.database.DSN.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size after vacuum: %v", err)
	}

	return &settingsv1.VacuumDatabaseResponse{
		SizeBeforeBytes: sizeBefore,
		SizeAfterBytes:  sizeAfter,
	}, nil
}

type pruneCounts struct {
	foregroundProcessesDeleted       int64
	foregroundProcessMetadataDeleted int64
	timelineDeleted                  int64
	timelineSemanticDocumentsDeleted int64
	timelineEmbeddingsDeleted        int64
	ledgerItemsDeleted               int64
	ledgerItemTimelineEntriesDeleted int64
	applicationsDeleted              int64
}

func pruneWindowFromRequest(req *settingsv1.PruneDatabaseRangeRequest) (time.Time, time.Time, error) {
	if req.GetStartedAt() == nil {
		return time.Time{}, time.Time{}, status.Error(codes.InvalidArgument, "started_at is required")
	}
	if req.GetEndedAt() == nil {
		return time.Time{}, time.Time{}, status.Error(codes.InvalidArgument, "ended_at is required")
	}
	if err := req.GetStartedAt().CheckValid(); err != nil {
		return time.Time{}, time.Time{}, status.Errorf(codes.InvalidArgument, "started_at is invalid: %v", err)
	}
	if err := req.GetEndedAt().CheckValid(); err != nil {
		return time.Time{}, time.Time{}, status.Errorf(codes.InvalidArgument, "ended_at is invalid: %v", err)
	}

	startedAt := req.GetStartedAt().AsTime().UTC()
	endedAt := req.GetEndedAt().AsTime().UTC()
	if !endedAt.After(startedAt) {
		return time.Time{}, time.Time{}, status.Error(codes.InvalidArgument, "ended_at must be after started_at")
	}
	return startedAt, endedAt, nil
}

func createPruneTempTables(ctx context.Context, tx *sql.Tx, startedAt time.Time, endedAt time.Time) error {
	statements := []struct {
		query string
		args  []any
	}{
		{query: "DROP TABLE IF EXISTS temp.prune_foreground_processes"},
		{query: "DROP TABLE IF EXISTS temp.prune_timeline"},
		{query: "DROP TABLE IF EXISTS temp.prune_timeline_semantic_documents"},
		{query: "DROP TABLE IF EXISTS temp.prune_ledger_items"},
		{query: "DROP TABLE IF EXISTS temp.prune_applications"},
		{query: "CREATE TEMP TABLE prune_foreground_processes (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_foreground_processes (id)
SELECT id
FROM foreground_processes
WHERE created_at_utc >= ? AND created_at_utc < ?`,
			args: []any{startedAt, endedAt},
		},
		// Timelines cascade-delete when either boundary foreground process is removed.
		{query: "CREATE TEMP TABLE prune_timeline (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_timeline (id)
SELECT id
FROM timeline
WHERE initial_foreground_process_id IN (SELECT id FROM prune_foreground_processes)
   OR end_foreground_process_id IN (SELECT id FROM prune_foreground_processes)`,
		},
		{query: "CREATE TEMP TABLE prune_timeline_semantic_documents (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_timeline_semantic_documents (id)
SELECT id
FROM timeline_semantic_documents
WHERE timeline_id IN (SELECT id FROM prune_timeline)`,
		},
		{query: "CREATE TEMP TABLE prune_ledger_items (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_ledger_items (id)
SELECT id
FROM ledger_items
WHERE started_at_utc >= ? AND started_at_utc < ?`,
			args: []any{startedAt, endedAt},
		},
		// Applications referenced by the pruned foreground processes (candidates for orphan cleanup).
		{query: "CREATE TEMP TABLE prune_applications (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_applications (id)
SELECT DISTINCT application_id
FROM foreground_processes
WHERE id IN (SELECT id FROM prune_foreground_processes)
  AND application_id IS NOT NULL`,
		},
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func deletePruneRows(ctx context.Context, tx *sql.Tx) (pruneCounts, error) {
	var counts pruneCounts
	var err error

	// Count the rows that ON DELETE CASCADE will remove before deleting the roots (RowsAffected only
	// reports the directly deleted rows, not cascaded ones).
	counts.foregroundProcessMetadataDeleted, err = execCount(ctx, tx, `
SELECT COUNT(*) FROM foreground_process_metadata
WHERE foreground_process_id IN (SELECT id FROM prune_foreground_processes)`)
	if err != nil {
		return counts, err
	}
	counts.timelineDeleted, err = execCount(ctx, tx, `SELECT COUNT(*) FROM prune_timeline`)
	if err != nil {
		return counts, err
	}
	counts.timelineSemanticDocumentsDeleted, err = execCount(ctx, tx, `SELECT COUNT(*) FROM prune_timeline_semantic_documents`)
	if err != nil {
		return counts, err
	}
	counts.timelineEmbeddingsDeleted, err = execCount(ctx, tx, `
SELECT COUNT(*) FROM timeline_embeddings
WHERE timeline_id IN (SELECT id FROM prune_timeline)
   OR timeline_semantic_documents_id IN (SELECT id FROM prune_timeline_semantic_documents)`)
	if err != nil {
		return counts, err
	}
	counts.ledgerItemTimelineEntriesDeleted, err = execCount(ctx, tx, `
SELECT COUNT(*) FROM ledger_item_timeline_entries
WHERE timeline_id IN (SELECT id FROM prune_timeline)
   OR ledger_items_id IN (SELECT id FROM prune_ledger_items)`)
	if err != nil {
		return counts, err
	}

	// ledger_items cascades to project_costs and ledger_item_timeline_entries.
	counts.ledgerItemsDeleted, err = execDelete(ctx, tx, `
DELETE FROM ledger_items
WHERE id IN (SELECT id FROM prune_ledger_items)`)
	if err != nil {
		return counts, err
	}

	// foreground_processes cascades to foreground_process_metadata, timeline, and transitively to
	// timeline_semantic_documents, timeline_embeddings and ledger_item_timeline_entries.
	counts.foregroundProcessesDeleted, err = execDelete(ctx, tx, `
DELETE FROM foreground_processes
WHERE id IN (SELECT id FROM prune_foreground_processes)`)
	if err != nil {
		return counts, err
	}

	counts.applicationsDeleted, err = execDelete(ctx, tx, `
DELETE FROM applications
WHERE id IN (SELECT id FROM prune_applications)
  AND NOT EXISTS (
    SELECT 1
    FROM foreground_processes
    WHERE foreground_processes.application_id = applications.id
  )`)
	if err != nil {
		return counts, err
	}

	return counts, nil
}

func execCount(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	var count int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func execDelete(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func dropPruneTempTables(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{
		"temp.prune_applications",
		"temp.prune_ledger_items",
		"temp.prune_timeline_semantic_documents",
		"temp.prune_timeline",
		"temp.prune_foreground_processes",
	} {
		if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return err
		}
	}
	return nil
}

func vacuumSQLiteDatabase(ctx context.Context, conn *sql.DB) error {
	if err := runWalCheckpointTruncate(ctx, conn); err != nil {
		return fmt.Errorf("checkpoint before vacuum: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	if err := runWalCheckpointTruncate(ctx, conn); err != nil {
		return fmt.Errorf("checkpoint after vacuum: %w", err)
	}
	return nil
}

func runWalCheckpointTruncate(ctx context.Context, conn *sql.DB) error {
	rows, err := conn.QueryContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var busy, logFrames, checkpointedFrames int64
		if err := rows.Scan(&busy, &logFrames, &checkpointedFrames); err != nil {
			return err
		}
		if busy != 0 {
			return fmt.Errorf("sqlite checkpoint busy: log=%d checkpointed=%d", logFrames, checkpointedFrames)
		}
	}
	return rows.Err()
}

func sqliteFootprintSize(dataSourceName string) (int64, error) {
	databasePath := sqliteDatabasePath(dataSourceName)
	if databasePath == "" || databasePath == ":memory:" {
		return 0, nil
	}

	var total int64
	for _, path := range []string{databasePath, databasePath + "-wal", databasePath + "-shm"} {
		info, err := os.Stat(path)
		if err == nil {
			total += info.Size()
			continue
		}
		if os.IsNotExist(err) {
			continue
		}
		return 0, fmt.Errorf("stat %s: %w", path, err)
	}
	return total, nil
}

func sqliteDatabasePath(dataSourceName string) string {
	base, _, _ := strings.Cut(dataSourceName, "?")
	if !strings.HasPrefix(base, "file:") {
		return base
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return strings.TrimPrefix(base, "file:")
	}
	if parsed.Opaque != "" {
		return parsed.Opaque
	}
	if parsed.Path != "" {
		return parsed.Path
	}
	return strings.TrimPrefix(base, "file:")
}
