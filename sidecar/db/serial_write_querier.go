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

func (s *SerialWriteQuerier) CreateProject(ctx context.Context, arg writequeries.CreateProjectParams) (writequeries.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateProject(ctx, arg)
}

func (s *SerialWriteQuerier) CreateSemanticDocumentEmbedding(ctx context.Context, arg writequeries.CreateSemanticDocumentEmbeddingParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateSemanticDocumentEmbedding(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTimesheetEntry(ctx context.Context, arg writequeries.CreateTimesheetEntryParams) (writequeries.TimesheetEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTimesheetEntry(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTimesheetEntryUsageBlock(ctx context.Context, arg writequeries.CreateTimesheetEntryUsageBlockParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTimesheetEntryUsageBlock(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTransitionEvent(ctx context.Context, arg writequeries.CreateTransitionEventParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTransitionEvent(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTransitionEventMetadata(ctx context.Context, arg writequeries.CreateTransitionEventMetadataParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTransitionEventMetadata(ctx, arg)
}

func (s *SerialWriteQuerier) CreateTransitionEventNow(ctx context.Context, arg writequeries.CreateTransitionEventNowParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.CreateTransitionEventNow(ctx, arg)
}

func (s *SerialWriteQuerier) DeleteProject(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteProject(ctx, id)
}

func (s *SerialWriteQuerier) DeleteSemanticDocumentEmbedding(ctx context.Context, semanticDocumentID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteSemanticDocumentEmbedding(ctx, semanticDocumentID)
}

func (s *SerialWriteQuerier) DeleteTimesheetEntry(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.DeleteTimesheetEntry(ctx, id)
}

func (s *SerialWriteQuerier) EnsureTimesheet(ctx context.Context, arg writequeries.EnsureTimesheetParams) (writequeries.Timesheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.EnsureTimesheet(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertAISettings(ctx context.Context, arg writequeries.UpsertAISettingsParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertAISettings(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertApplication(ctx context.Context, arg writequeries.UpsertApplicationParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertApplication(ctx, arg)
}

func (s *SerialWriteQuerier) UpsertSemanticDocument(ctx context.Context, arg writequeries.UpsertSemanticDocumentParams) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.querier.UpsertSemanticDocument(ctx, arg)
}
