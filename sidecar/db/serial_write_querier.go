package db

import (
	"context"
	"database/sql"
	"sync"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
)

// SerialWriteQuerier owns the write connection and a mutex so every write executes serially. SQLite
// allows only a single writer; acquiring the lock in-process before issuing the statement keeps
// SQLite from having to arbitrate concurrent writers (and throwing SQLITE_BUSY / busy timeouts). The
// same mutex guards the generated write methods, WriteTx, and WithWriteConn, so single-statement
// writes, multi-statement transactions, and raw maintenance (VACUUM/prune) all serialize against one
// another. conn is unexported: raw write access is only reachable under the lock via WithWriteConn.
type SerialWriteQuerier struct {
	querier writequeries.Querier
	conn    *sql.DB
	mu      sync.Mutex
}

func newSerialWriteQuerier(conn *sql.DB) *SerialWriteQuerier {
	return &SerialWriteQuerier{querier: writequeries.New(conn), conn: conn}
}

var _ writequeries.Querier = (*SerialWriteQuerier)(nil)

// WriteTx runs fn inside a write transaction while holding the write lock for its full duration, so
// the whole transaction is serialized against all other writes. The transaction is rolled back if fn
// returns an error (or panics) and committed otherwise.
func (s *SerialWriteQuerier) WriteTx(ctx context.Context, fn func(*writequeries.Queries) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(writequeries.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// WithWriteConn runs fn with the write connection while holding the write lock, for serialized writes
// that can't go through the generated queriers or WriteTx (e.g. VACUUM, which SQLite forbids inside a
// transaction, and the prune temp-table workflow that issues raw SQL).
func (s *SerialWriteQuerier) WithWriteConn(fn func(*sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s.conn)
}

func (s *SerialWriteQuerier) CreateForegroundProcessMetadata(ctx context.Context, arg writequeries.CreateForegroundProcessMetadataParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateForegroundProcessMetadata(ctx, arg)
}

func (s *SerialWriteQuerier) CreateLedgerItem(ctx context.Context, arg writequeries.CreateLedgerItemParams) (writequeries.LedgerItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateLedgerItem(ctx, arg)
}

func (s *SerialWriteQuerier) CreateLedgerItemTimelineEntry(ctx context.Context, arg writequeries.CreateLedgerItemTimelineEntryParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateLedgerItemTimelineEntry(ctx, arg)
}

func (s *SerialWriteQuerier) CreateProject(ctx context.Context, name string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateProject(ctx, name)
}

func (s *SerialWriteQuerier) CreateProjectCost(ctx context.Context, arg writequeries.CreateProjectCostParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateProjectCost(ctx, arg)
}

func (s *SerialWriteQuerier) CreateProjectDetails(ctx context.Context, arg writequeries.CreateProjectDetailsParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateProjectDetails(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTimeline(ctx context.Context, arg writequeries.CreateTimelineParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTimeline(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTimelineEmbedding(ctx context.Context, arg writequeries.CreateTimelineEmbeddingParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTimelineEmbedding(ctx, arg)
}

func (s *SerialWriteQuerier) DeleteLedgerItem(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteLedgerItem(ctx, id)
}

func (s *SerialWriteQuerier) DeleteProject(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteProject(ctx, id)
}

func (s *SerialWriteQuerier) DeleteTimelineEmbedding(ctx context.Context, timelineSemanticDocumentsID sql.NullInt64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteTimelineEmbedding(ctx, timelineSemanticDocumentsID)
}

func (s *SerialWriteQuerier) EnsureLedger(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.EnsureLedger(ctx)
}

func (s *SerialWriteQuerier) UpsertApplication(ctx context.Context, arg writequeries.UpsertApplicationParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertApplication(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertApplicationSettings(ctx context.Context, arg writequeries.UpsertApplicationSettingsParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertApplicationSettings(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertForegroundProcess(ctx context.Context, arg writequeries.UpsertForegroundProcessParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertForegroundProcess(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertTimelineSemanticDocument(ctx context.Context, arg writequeries.UpsertTimelineSemanticDocumentParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertTimelineSemanticDocument(ctx, arg)
}
