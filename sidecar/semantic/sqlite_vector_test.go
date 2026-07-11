package semantic

import (
	"context"
	"database/sql"
	"testing"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
)

func TestSQLiteVectorStoreQuantizedScanMatchesExactNearestNeighbor(t *testing.T) {
	ctx := context.Background()
	database := newSemanticVectorTestDatabase(t, ctx)

	firstID := createSemanticVectorDocument(t, ctx, database.WriteQuerier, "doc:first", []float32{1, 0, 0, 0})
	secondID := createSemanticVectorDocument(t, ctx, database.WriteQuerier, "doc:second", []float32{0, 1, 0, 0})
	firstEmbeddingRowID := semanticEmbeddingRowID(t, ctx, database.ReadConn, firstID)
	secondEmbeddingRowID := semanticEmbeddingRowID(t, ctx, database.ReadConn, secondID)

	store := NewSQLiteVectorStore(database.ReadConn)
	if err := store.Check(ctx); err != nil {
		t.Fatalf("check sqlite-vector store: %v", err)
	}
	if err := store.EnsureQuantized(ctx); err != nil {
		t.Fatalf("ensure sqlite-vector quantized: %v", err)
	}

	query, err := EncodeFloat32Vector(paddedSemanticTestVector([]float32{1, 0, 0, 0}))
	if err != nil {
		t.Fatalf("encode query: %v", err)
	}

	exactRowID := nearestVectorRowID(t, ctx, database.ReadConn, "vector_full_scan", query)
	quantizedRowID := nearestVectorRowID(t, ctx, database.ReadConn, "vector_quantize_scan", query)

	if exactRowID != firstEmbeddingRowID {
		t.Fatalf("expected exact nearest row %d, got %d; second row was %d", firstEmbeddingRowID, exactRowID, secondEmbeddingRowID)
	}
	if quantizedRowID != exactRowID {
		t.Fatalf("expected quantized nearest row to match exact row %d, got %d", exactRowID, quantizedRowID)
	}
}

func createSemanticVectorDocument(t *testing.T, ctx context.Context, q writequeries.Querier, key string, values []float32) int64 {
	t.Helper()
	documentID, err := q.UpsertTimelineSemanticDocument(ctx, writequeries.UpsertTimelineSemanticDocumentParams{
		DocumentKey: key,
		Type:        sql.NullInt64{Int64: 1, Valid: true},
		Content:     key,
	})
	if err != nil {
		t.Fatalf("upsert semantic document: %v", err)
	}
	encoded, err := EncodeFloat32Vector(paddedSemanticTestVector(values))
	if err != nil {
		t.Fatalf("encode embedding: %v", err)
	}
	if err := q.CreateTimelineEmbedding(ctx, writequeries.CreateTimelineEmbeddingParams{
		TimelineSemanticDocumentsID: sql.NullInt64{Int64: documentID, Valid: true},
		EmbeddingModelID:            sql.NullInt64{Int64: 1, Valid: true},
		Dimension:                   StoreEmbeddingDimension,
		Embedding:                   encoded,
	}); err != nil {
		t.Fatalf("create semantic document embedding: %v", err)
	}
	return documentID
}

func paddedSemanticTestVector(values []float32) []float32 {
	padded := make([]float32, StoreEmbeddingDimension)
	copy(padded, values)
	return padded
}

func semanticEmbeddingRowID(t *testing.T, ctx context.Context, conn *sql.DB, documentID int64) int64 {
	t.Helper()
	var rowID int64
	if err := conn.QueryRowContext(ctx, `
SELECT rowid
FROM timeline_embeddings
WHERE timeline_semantic_documents_id = ?`, documentID).Scan(&rowID); err != nil {
		t.Fatalf("read semantic embedding rowid: %v", err)
	}
	return rowID
}

func nearestVectorRowID(t *testing.T, ctx context.Context, conn *sql.DB, functionName string, query []byte) int64 {
	t.Helper()
	var rowID int64
	sql := "SELECT rowid FROM " + functionName + "('timeline_embeddings', 'embedding', ?, 1)"
	if err := conn.QueryRowContext(ctx, sql, query).Scan(&rowID); err != nil {
		t.Fatalf("read nearest vector rowid using %s: %v", functionName, err)
	}
	return rowID
}
