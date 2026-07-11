package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
)

// writeTxRunner runs a function inside a serialized write transaction. *db.Database satisfies it.
type writeTxRunner interface {
	WriteTx(ctx context.Context, fn func(*writequeries.Queries) error) error
}

type Indexer struct {
	writeTx          writeTxRunner
	readQuerier      readqueries.Querier
	embedder         Embedder
	embeddingModelID int64
	location         *time.Location
}

func NewIndexer(writeTx writeTxRunner, readQuerier readqueries.Querier, embedder Embedder, embeddingModelID int64) *Indexer {
	return &Indexer{writeTx: writeTx, readQuerier: readQuerier, embedder: embedder, embeddingModelID: embeddingModelID, location: time.Local}
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
	rows, err := i.readQuerier.ListTransitionEventDocumentSourcesForWindow(ctx, readqueries.ListTransitionEventDocumentSourcesForWindowParams{
		WindowStartedAt: dayStart.UTC(),
		WindowEndedAt:   dayEnd.UTC(),
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

	var documentID int64
	if err := i.writeTx.WriteTx(ctx, func(q *writequeries.Queries) error {
		documentID, err = q.UpsertTimelineSemanticDocument(ctx, writequeries.UpsertTimelineSemanticDocumentParams{
			DocumentKey: spec.Key,
			TimelineID:  spec.TransitionEventID,
			Type:        documentTypeID(spec.Type),
			Content:     content,
		})
		if err != nil {
			return fmt.Errorf("upsert semantic document %q: %w", spec.Key, err)
		}

		if err := q.DeleteTimelineEmbedding(ctx, sql.NullInt64{Int64: documentID, Valid: true}); err != nil {
			return fmt.Errorf("delete semantic document embedding %q: %w", spec.Key, err)
		}

		if err := q.CreateTimelineEmbedding(ctx, writequeries.CreateTimelineEmbeddingParams{
			TimelineID:                  spec.TransitionEventID,
			TimelineSemanticDocumentsID: sql.NullInt64{Int64: documentID, Valid: true},
			EmbeddingModelID:            sql.NullInt64{Int64: i.embeddingModelID, Valid: true},
			Dimension:                   int64(i.embedder.Dimension()),
			Embedding:                   encoded,
		}); err != nil {
			return fmt.Errorf("create semantic document embedding %q: %w", spec.Key, err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return documentID, nil
}
