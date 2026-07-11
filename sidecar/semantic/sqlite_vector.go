package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

const (
	sqliteVectorEmbeddingTable  = "timeline_embeddings"
	sqliteVectorEmbeddingColumn = "embedding"
	sqliteVectorOptions         = "dimension=4096,type=FLOAT32,distance=COSINE"
	sqliteVectorQuantizeOptions = "qtype=TURBO,qbits=4"
)

type SQLiteVectorStore struct {
	conn               *sql.DB
	mu                 sync.Mutex
	initialized        bool
	quantizedSignature string
}

func NewSQLiteVectorStore(conn *sql.DB) *SQLiteVectorStore {
	return &SQLiteVectorStore{conn: conn}
}

func (s *SQLiteVectorStore) Check(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initializeLocked(ctx)
}

func (s *SQLiteVectorStore) EnsureQuantized(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.initializeLocked(ctx); err != nil {
		return err
	}

	signature, rowCount, err := s.embeddingSignatureLocked(ctx)
	if err != nil {
		return err
	}
	if rowCount == 0 || signature == s.quantizedSignature {
		s.quantizedSignature = signature
		return nil
	}

	var quantizedRows int64
	if err := s.conn.QueryRowContext(
		ctx,
		"SELECT vector_quantize(?, ?, ?)",
		sqliteVectorEmbeddingTable,
		sqliteVectorEmbeddingColumn,
		sqliteVectorQuantizeOptions,
	).Scan(&quantizedRows); err != nil {
		return fmt.Errorf("quantize sqlite-vector embeddings: %w", err)
	}
	if _, err := s.conn.ExecContext(
		ctx,
		"SELECT vector_quantize_preload(?, ?)",
		sqliteVectorEmbeddingTable,
		sqliteVectorEmbeddingColumn,
	); err != nil {
		return fmt.Errorf("preload sqlite-vector embeddings: %w", err)
	}

	s.quantizedSignature = signature
	return nil
}

func (s *SQLiteVectorStore) initializeLocked(ctx context.Context) error {
	if s.conn == nil {
		return fmt.Errorf("sqlite connection is required")
	}
	if s.initialized {
		return nil
	}

	var version string
	if err := s.conn.QueryRowContext(ctx, "SELECT vector_version()").Scan(&version); err != nil {
		return fmt.Errorf("sqlite-vector extension is unavailable: %w", err)
	}
	if _, err := s.conn.ExecContext(
		ctx,
		"SELECT vector_init(?, ?, ?)",
		sqliteVectorEmbeddingTable,
		sqliteVectorEmbeddingColumn,
		sqliteVectorOptions,
	); err != nil {
		return fmt.Errorf("initialize sqlite-vector embeddings: %w", err)
	}

	s.initialized = true
	return nil
}

func (s *SQLiteVectorStore) embeddingSignatureLocked(ctx context.Context) (string, int64, error) {
	var rowCount int64
	var latestRowID int64
	if err := s.conn.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(MAX(id), 0)
FROM timeline_embeddings`).Scan(&rowCount, &latestRowID); err != nil {
		return "", 0, fmt.Errorf("read semantic embedding signature: %w", err)
	}
	return fmt.Sprintf("%d:%d", rowCount, latestRowID), rowCount, nil
}
