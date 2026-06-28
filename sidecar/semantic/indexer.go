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
	location    *time.Location
}

func NewIndexer(writeConn *sql.DB, readQuerier queries.Querier, embedder Embedder) *Indexer {
	return &Indexer{writeConn: writeConn, readQuerier: readQuerier, embedder: embedder, location: time.Local}
}

func (i *Indexer) IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error) {
	if i.embedder == nil {
		return 0, fmt.Errorf("embedder is required")
	}

	src, err := i.readQuerier.GetSemanticEventDocumentSource(ctx, transitionEventID)
	if err != nil {
		return 0, fmt.Errorf("get transition event document source: %w", err)
	}

	spec := eventDocumentSpec(src, i.location)
	documentID, err := i.upsertEmbeddedDocument(ctx, spec)
	if err != nil {
		return 0, fmt.Errorf("index semantic event document: %w", err)
	}

	if err := i.RefreshSummariesForTime(ctx, src.StartedAt); err != nil {
		return 0, fmt.Errorf("refresh semantic summaries: %w", err)
	}

	return documentID, nil
}

func (i *Indexer) RefreshSummariesForTime(ctx context.Context, value time.Time) error {
	if i.embedder == nil {
		return fmt.Errorf("embedder is required")
	}
	if i.location == nil {
		i.location = time.Local
	}

	dayStart := startOfLocalDay(value, i.location)
	dayEnd := dayStart.AddDate(0, 0, 1)
	rows, err := i.readQuerier.ListTransitionEventDocumentSourcesForWindow(ctx, queries.ListTransitionEventDocumentSourcesForWindowParams{
		WindowStartedAt: dayStart,
		WindowEndedAt:   dayEnd,
	})
	if err != nil {
		return fmt.Errorf("list transition event document sources for semantic summaries: %w", err)
	}

	for _, spec := range summaryDocumentSpecs(rows, dayStart, i.location) {
		if _, err := i.upsertEmbeddedDocument(ctx, spec); err != nil {
			return err
		}
	}

	return nil
}

func (i *Indexer) upsertEmbeddedDocument(ctx context.Context, spec DocumentSpec) (int64, error) {
	content := spec.Content
	embedding, err := i.embedder.Embed(ctx, content)
	if err != nil {
		return 0, fmt.Errorf("embed semantic document %q: %w", spec.Key, err)
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

	documentID, err := q.UpsertSemanticDocument(ctx, queries.UpsertSemanticDocumentParams{
		DocumentKey:        spec.Key,
		DocumentType:       spec.Type,
		TransitionEventID:  spec.TransitionEventID,
		StartedAt:          spec.StartedAt,
		EndedAt:            spec.EndedAt,
		Content:            content,
		EmbeddingModel:     i.embedder.Model(),
		EmbeddingDimension: int64(i.embedder.Dimension()),
		EmbeddedAt:         sql.NullTime{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("upsert semantic document %q: %w", spec.Key, err)
	}

	if err := q.DeleteSemanticDocumentEmbedding(ctx, documentID); err != nil {
		return 0, fmt.Errorf("delete semantic document embedding %q: %w", spec.Key, err)
	}

	if err := q.CreateSemanticDocumentEmbedding(ctx, queries.CreateSemanticDocumentEmbeddingParams{
		SemanticDocumentID: documentID,
		Embedding:          encoded,
	}); err != nil {
		return 0, fmt.Errorf("create semantic document embedding %q: %w", spec.Key, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit semantic index transaction: %w", err)
	}

	return documentID, nil
}
