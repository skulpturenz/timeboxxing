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

	size, err := sqliteFootprintSize(s.database.DataSourceName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size: %v", err)
	}
	return &settingsv1.DatabaseMaintenanceStatus{SizeBytes: size}, nil
}

func (s *Server) PruneDatabaseRange(ctx context.Context, req *settingsv1.PruneDatabaseRangeRequest) (*settingsv1.PruneDatabaseRangeResponse, error) {
	if s.database == nil || s.database.WriteConn == nil {
		return nil, status.Error(codes.FailedPrecondition, "database is unavailable")
	}

	startedAt, endedAt, err := pruneWindowFromRequest(req)
	if err != nil {
		return nil, err
	}

	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	tx, err := s.database.WriteConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin prune transaction: %v", err)
	}
	defer tx.Rollback()

	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_ = dropPruneTempTables(ctx, tx)
		}
	}()

	if err := createPruneTempTables(ctx, tx, startedAt, endedAt); err != nil {
		return nil, status.Errorf(codes.Internal, "prepare prune scope: %v", err)
	}

	counts, err := deletePruneRows(ctx, tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete pruned rows: %v", err)
	}

	if err := dropPruneTempTables(ctx, tx); err != nil {
		return nil, status.Errorf(codes.Internal, "clean prune scope: %v", err)
	}
	cleanupNeeded = false

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit prune transaction: %v", err)
	}

	size, err := sqliteFootprintSize(s.database.DataSourceName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload database size: %v", err)
	}

	return &settingsv1.PruneDatabaseRangeResponse{
		SizeBytes:                 size,
		TimesheetEntriesDeleted:   counts.timesheetEntriesDeleted,
		UsageLinksDeleted:         counts.usageLinksDeleted,
		TimesheetsDeleted:         counts.timesheetsDeleted,
		TransitionEventsDeleted:   counts.transitionEventsDeleted,
		TransitionMetadataDeleted: counts.transitionMetadataDeleted,
		SemanticDocumentsDeleted:  counts.semanticDocumentsDeleted,
		EmbeddingsDeleted:         counts.embeddingsDeleted,
		ApplicationsDeleted:       counts.applicationsDeleted,
	}, nil
}

func (s *Server) VacuumDatabase(ctx context.Context, _ *settingsv1.VacuumDatabaseRequest) (*settingsv1.VacuumDatabaseResponse, error) {
	if s.database == nil || s.database.WriteConn == nil {
		return nil, status.Error(codes.FailedPrecondition, "database is unavailable")
	}

	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	sizeBefore, err := sqliteFootprintSize(s.database.DataSourceName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size before vacuum: %v", err)
	}

	if err := vacuumSQLiteDatabase(ctx, s.database.WriteConn); err != nil {
		return nil, status.Errorf(codes.Internal, "vacuum database: %v", err)
	}

	sizeAfter, err := sqliteFootprintSize(s.database.DataSourceName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get database size after vacuum: %v", err)
	}

	return &settingsv1.VacuumDatabaseResponse{
		SizeBeforeBytes: sizeBefore,
		SizeAfterBytes:  sizeAfter,
	}, nil
}

type pruneCounts struct {
	timesheetEntriesDeleted   int64
	usageLinksDeleted         int64
	timesheetsDeleted         int64
	transitionEventsDeleted   int64
	transitionMetadataDeleted int64
	semanticDocumentsDeleted  int64
	embeddingsDeleted         int64
	applicationsDeleted       int64
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
		{query: "DROP TABLE IF EXISTS temp.prune_transition_events"},
		{query: "DROP TABLE IF EXISTS temp.prune_applications"},
		{query: "DROP TABLE IF EXISTS temp.prune_semantic_documents"},
		{query: "DROP TABLE IF EXISTS temp.prune_timesheets"},
		{query: "DROP TABLE IF EXISTS temp.prune_timesheet_entries"},
		{query: "CREATE TEMP TABLE prune_transition_events (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_transition_events (id)
SELECT id
FROM transition_events
WHERE ended_at > ? AND started_at < ?`,
			args: []any{startedAt, endedAt},
		},
		{query: "CREATE TEMP TABLE prune_applications (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_applications (id)
SELECT DISTINCT application_id
FROM transition_events
WHERE id IN (SELECT id FROM prune_transition_events)
  AND application_id IS NOT NULL`,
		},
		{query: "CREATE TEMP TABLE prune_semantic_documents (id INTEGER PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_semantic_documents (id)
SELECT id
FROM semantic_documents
WHERE transition_event_id IN (SELECT id FROM prune_transition_events)
   OR (
     started_at IS NOT NULL
     AND ended_at IS NOT NULL
     AND ended_at > ?
     AND started_at < ?
   )`,
			args: []any{startedAt, endedAt},
		},
		{query: "CREATE TEMP TABLE prune_timesheets (id TEXT PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_timesheets (id)
SELECT id
FROM timesheets
WHERE ended_at > ? AND started_at < ?`,
			args: []any{startedAt, endedAt},
		},
		{query: "CREATE TEMP TABLE prune_timesheet_entries (id TEXT PRIMARY KEY)"},
		{
			query: `
INSERT INTO prune_timesheet_entries (id)
SELECT id
FROM timesheet_entries
WHERE timesheet_id IN (SELECT id FROM prune_timesheets)`,
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

	counts.embeddingsDeleted, err = execDelete(ctx, tx, `
DELETE FROM semantic_document_embeddings
WHERE semantic_document_id IN (SELECT id FROM prune_semantic_documents)`)
	if err != nil {
		return counts, err
	}

	counts.semanticDocumentsDeleted, err = execDelete(ctx, tx, `
DELETE FROM semantic_documents
WHERE id IN (SELECT id FROM prune_semantic_documents)`)
	if err != nil {
		return counts, err
	}

	counts.transitionMetadataDeleted, err = execDelete(ctx, tx, `
DELETE FROM transition_event_metadata
WHERE transition_event_id IN (SELECT id FROM prune_transition_events)`)
	if err != nil {
		return counts, err
	}

	counts.usageLinksDeleted, err = execDelete(ctx, tx, `
DELETE FROM timesheet_entry_usage_blocks
WHERE timesheet_entry_id IN (SELECT id FROM prune_timesheet_entries)
   OR usage_id IN (SELECT 'sidecar-' || id FROM prune_transition_events)`)
	if err != nil {
		return counts, err
	}

	counts.timesheetEntriesDeleted, err = execDelete(ctx, tx, `
DELETE FROM timesheet_entries
WHERE id IN (SELECT id FROM prune_timesheet_entries)`)
	if err != nil {
		return counts, err
	}

	counts.timesheetsDeleted, err = execDelete(ctx, tx, `
DELETE FROM timesheets
WHERE id IN (SELECT id FROM prune_timesheets)
  AND NOT EXISTS (
    SELECT 1
    FROM timesheet_entries
    WHERE timesheet_entries.timesheet_id = timesheets.id
  )`)
	if err != nil {
		return counts, err
	}

	counts.transitionEventsDeleted, err = execDelete(ctx, tx, `
DELETE FROM transition_events
WHERE id IN (SELECT id FROM prune_transition_events)`)
	if err != nil {
		return counts, err
	}

	counts.applicationsDeleted, err = execDelete(ctx, tx, `
DELETE FROM applications
WHERE id IN (SELECT id FROM prune_applications)
  AND NOT EXISTS (
  SELECT 1
  FROM transition_events
  WHERE transition_events.application_id = applications.id
)`)
	if err != nil {
		return counts, err
	}

	return counts, nil
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
		"temp.prune_timesheet_entries",
		"temp.prune_timesheets",
		"temp.prune_semantic_documents",
		"temp.prune_applications",
		"temp.prune_transition_events",
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
