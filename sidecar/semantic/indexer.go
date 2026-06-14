package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type Indexer struct {
	writeConn   *sql.DB
	readQuerier queries.Querier
	embedder    Embedder
}

func NewIndexer(writeConn *sql.DB, readQuerier queries.Querier, embedder Embedder) *Indexer {
	return &Indexer{writeConn: writeConn, readQuerier: readQuerier, embedder: embedder}
}

func (i *Indexer) IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error) {
	if i.embedder == nil {
		return 0, fmt.Errorf("embedder is required")
	}

	src, err := i.readQuerier.GetTransitionEventDocumentSource(ctx, transitionEventID)
	if err != nil {
		return 0, fmt.Errorf("get transition event document source: %w", err)
	}

	content := RenderTransitionEventDocument(src)
	embedding, err := i.embedder.Embed(ctx, content)
	if err != nil {
		return 0, fmt.Errorf("embed transition event document: %w", err)
	}
	if len(embedding) != i.embedder.Dimension() {
		return 0, fmt.Errorf("embedding dimension mismatch: got %d, want %d", len(embedding), i.embedder.Dimension())
	}
	encoded, err := EncodeFloat32Vector(embedding)
	if err != nil {
		return 0, err
	}

	tx, err := i.writeConn.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin semantic index transaction: %w", err)
	}
	defer tx.Rollback()

	q := queries.New(tx)

	documentID, err := q.CreateTransitionEventDocument(ctx, queries.CreateTransitionEventDocumentParams{
		TransitionEventID:  transitionEventID,
		Content:            content,
		EmbeddingModel:     i.embedder.Model(),
		EmbeddingDimension: int64(i.embedder.Dimension()),
		EmbeddedAt:         sql.NullTime{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("create transition event document: %w", err)
	}

	if err := q.CreateTransitionEventDocumentEmbedding(ctx, queries.CreateTransitionEventDocumentEmbeddingParams{
		TransitionEventDocumentID: documentID,
		Embedding:                 encoded,
	}); err != nil {
		return 0, fmt.Errorf("create transition event document embedding: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit semantic index transaction: %w", err)
	}

	return documentID, nil
}
