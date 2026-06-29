package semantic

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type modelKeyEmbedder struct {
	model string
}

func (e modelKeyEmbedder) Model() string { return e.model }

func (modelKeyEmbedder) Dimension() int { return StoreEmbeddingDimension }

func (modelKeyEmbedder) Embed(context.Context, string) ([]float32, error) {
	values := make([]float32, StoreEmbeddingDimension)
	values[0] = 1
	return values, nil
}

func TestListMissingSemanticEventDocumentIDsFindsMissingDocumentsAndEmbeddings(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	indexer := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{})

	completeID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	missingEmbeddingID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	missingDocumentID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC))

	if _, err := indexer.IndexTransitionEvent(ctx, completeID); err != nil {
		t.Fatalf("index complete event: %v", err)
	}
	missingEmbeddingDocumentID, err := indexer.IndexTransitionEvent(ctx, missingEmbeddingID)
	if err != nil {
		t.Fatalf("index event missing embedding: %v", err)
	}
	if err := database.WriteQuerier.DeleteSemanticDocumentEmbedding(ctx, missingEmbeddingDocumentID); err != nil {
		t.Fatalf("delete embedding: %v", err)
	}

	ids, err := database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, queries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: fakeEmbedder{}.Model(),
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("list missing semantic event document ids: %v", err)
	}

	want := []int64{missingDocumentID, missingEmbeddingID}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("expected missing ids %v, got %v", want, ids)
	}
}

func TestBackfillerIndexesMissingTransitionEvents(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	indexer := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{})

	firstID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	secondID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	if _, err := indexer.IndexTransitionEvent(ctx, firstID); err != nil {
		t.Fatalf("index first event: %v", err)
	}

	result, err := NewBackfiller(database.ReadQuerier, indexer, fakeEmbedder{}.Model()).BackfillMissing(ctx, 10)
	if err != nil {
		t.Fatalf("backfill missing transition events: %v", err)
	}
	if result != (BackfillResult{Checked: 1, Indexed: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}

	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "semantic_documents", "document_type = 'event'"); count != 2 {
		t.Fatalf("expected two semantic event documents, got %d", count)
	}
	if count := countSemanticRowsWhere(t, ctx, database.ReadConn, "semantic_document_float32_embeddings", "semantic_document_id IN (SELECT id FROM semantic_documents WHERE document_type = 'event')"); count != 2 {
		t.Fatalf("expected two semantic event embeddings, got %d", count)
	}

	ids, err := database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, queries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: fakeEmbedder{}.Model(),
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("list missing semantic event document ids: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no missing ids after backfill, got %v", ids)
	}
	if firstID == secondID {
		t.Fatal("expected distinct transition events")
	}
}

func TestBackfillerTreatsEmbeddingModelChangesAsMissing(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	firstModel := modelKeyEmbedder{model: "embedding-model-a"}
	secondModel := modelKeyEmbedder{model: "embedding-model-b"}

	eventID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	if _, err := NewIndexer(database.WriteConn, database.ReadQuerier, firstModel).IndexTransitionEvent(ctx, eventID); err != nil {
		t.Fatalf("index first model: %v", err)
	}

	ids, err := database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, queries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: secondModel.Model(),
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("list missing for second model: %v", err)
	}
	if !reflect.DeepEqual(ids, []int64{eventID}) {
		t.Fatalf("expected event to be missing for second model, got %v", ids)
	}

	result, err := NewBackfiller(
		database.ReadQuerier,
		NewIndexer(database.WriteConn, database.ReadQuerier, secondModel),
		secondModel.Model(),
	).BackfillMissing(ctx, 10)
	if err != nil {
		t.Fatalf("backfill second model: %v", err)
	}
	if result != (BackfillResult{Checked: 1, Indexed: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	ids, err = database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, queries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: secondModel.Model(),
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("list missing after second model backfill: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no missing ids for second model after backfill, got %v", ids)
	}

	ids, err = database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, queries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModel: firstModel.Model(),
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("list missing after old model replacement: %v", err)
	}
	if !reflect.DeepEqual(ids, []int64{eventID}) {
		t.Fatalf("expected old model embedding to be replaced, got missing ids %v", ids)
	}
}

func TestBackfillerHasMissing(t *testing.T) {
	ctx := context.Background()

	hasMissing, err := NewBackfiller(staticMissingTransitionEventLister{ids: []int64{42}}, nil, fakeEmbedder{}.Model()).HasMissing(ctx)
	if err != nil {
		t.Fatalf("has missing: %v", err)
	}
	if !hasMissing {
		t.Fatal("expected missing transition event documents")
	}

	hasMissing, err = NewBackfiller(staticMissingTransitionEventLister{}, nil, fakeEmbedder{}.Model()).HasMissing(ctx)
	if err != nil {
		t.Fatalf("has missing empty: %v", err)
	}
	if hasMissing {
		t.Fatal("expected no missing transition event documents")
	}
}

func TestBackfillerContinuesAfterIndexFailure(t *testing.T) {
	ctx := context.Background()
	indexer := &recordingBackfillIndexer{failID: 2}
	result, err := NewBackfiller(staticMissingTransitionEventLister{ids: []int64{1, 2, 3}}, indexer, fakeEmbedder{}.Model()).BackfillMissing(ctx, 10)
	if err == nil {
		t.Fatal("expected partial backfill error")
	}

	if result != (BackfillResult{Checked: 3, Indexed: 2, Failed: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	wantIndexedIDs := []int64{1, 2, 3}
	if !reflect.DeepEqual(indexer.ids, wantIndexedIDs) {
		t.Fatalf("expected attempted ids %v, got %v", wantIndexedIDs, indexer.ids)
	}
}

func TestBackfillCoordinatorStoresPartialFailure(t *testing.T) {
	ctx := context.Background()
	coordinator := NewBackfillCoordinator(
		ctx,
		NewBackfiller(
			staticMissingTransitionEventLister{ids: []int64{1, 2, 3}},
			&recordingBackfillIndexer{failID: 2},
			fakeEmbedder{}.Model(),
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if !coordinator.Start(10) {
		t.Fatal("expected backfill to start")
	}
	deadline := time.Now().Add(time.Second)
	for {
		status := coordinator.Status()
		if !status.Running {
			if status.LastResult != (BackfillResult{Checked: 3, Indexed: 2, Failed: 1}) {
				t.Fatalf("unexpected final status: %+v", status)
			}
			if status.LastError != "Semantic indexing failed. Check sidecar logs." {
				t.Fatalf("unexpected last error %q", status.LastError)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for backfill completion: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBackfillCoordinatorRunsOneBackfillAtATime(t *testing.T) {
	ctx := context.Background()
	started := make(chan struct{})
	release := make(chan struct{})
	indexer := &blockingBackfillIndexer{
		started: started,
		release: release,
	}
	coordinator := NewBackfillCoordinator(
		ctx,
		NewBackfiller(staticMissingTransitionEventLister{ids: []int64{1}}, indexer, fakeEmbedder{}.Model()),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if !coordinator.Start(10) {
		t.Fatal("expected first backfill to start")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for backfill to start")
	}
	if coordinator.Start(10) {
		t.Fatal("expected duplicate backfill start to be ignored")
	}

	close(release)
	deadline := time.Now().Add(time.Second)
	for {
		status := coordinator.Status()
		if !status.Running {
			if status.LastResult != (BackfillResult{Checked: 1, Indexed: 1}) {
				t.Fatalf("unexpected final status: %+v", status)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for backfill completion: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type staticMissingTransitionEventLister struct {
	ids []int64
	err error
}

func (s staticMissingTransitionEventLister) ListMissingSemanticEventDocumentIDs(context.Context, queries.ListMissingSemanticEventDocumentIDsParams) ([]int64, error) {
	return s.ids, s.err
}

type recordingBackfillIndexer struct {
	ids    []int64
	failID int64
}

func (r *recordingBackfillIndexer) IndexTransitionEvent(_ context.Context, transitionEventID int64) (int64, error) {
	r.ids = append(r.ids, transitionEventID)
	if transitionEventID == r.failID {
		return 0, fmt.Errorf("index failed")
	}
	return transitionEventID, nil
}

type blockingBackfillIndexer struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (b *blockingBackfillIndexer) IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error) {
	close(b.started)
	select {
	case <-b.release:
		return transitionEventID, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
