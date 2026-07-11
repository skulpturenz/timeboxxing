package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
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
	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1)

	completeID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	missingEmbeddingID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	missingDocumentID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC))

	if _, err := indexer.IndexTransitionEvent(ctx, completeID); err != nil {
		t.Fatalf("index complete event: %v", err)
	}
	missingEmbeddingDocumentID, err := indexer.IndexTransitionEvent(ctx, missingEmbeddingID)
	if err != nil {
		t.Fatalf("index event missing embedding: %v", err)
	}
	if err := database.WriteQuerier.DeleteTimelineEmbedding(ctx, sql.NullInt64{Int64: missingEmbeddingDocumentID, Valid: true}); err != nil {
		t.Fatalf("delete embedding: %v", err)
	}

	ids, err := database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, readqueries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModelID: sql.NullInt64{Int64: 1, Valid: true},
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("list missing semantic event document ids: %v", err)
	}

	want := []int64{missingDocumentID, missingEmbeddingID}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("expected missing ids %v, got %v", want, ids)
	}
}

func TestBackfillerEnqueuesMissingTransitionEvents(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	indexer := NewIndexer(database.WriteQuerier, database.ReadQuerier, fakeEmbedder{}, 1)
	enqueuer := &recordingBackfillEnqueuer{}

	firstID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	secondID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	if _, err := indexer.IndexTransitionEvent(ctx, firstID); err != nil {
		t.Fatalf("index first event: %v", err)
	}

	result, err := NewBackfiller(database.ReadQuerier, enqueuer, 1).BackfillMissing(ctx, 10)
	if err != nil {
		t.Fatalf("backfill missing transition events: %v", err)
	}
	if result != (BackfillResult{Checked: 1, Enqueued: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	if !reflect.DeepEqual(enqueuer.ids, []int64{secondID}) {
		t.Fatalf("expected enqueued ids %v, got %v", []int64{secondID}, enqueuer.ids)
	}
	if firstID == secondID {
		t.Fatal("expected distinct transition events")
	}
}

func TestBackfillerTreatsEmbeddingModelChangesAsMissing(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	// Embedding models are identified by their seeded models.id now (1-3 are embedding models); the
	// embedder's Model() label is unrelated to the id used to scope the index.
	const firstModelID int64 = 1
	const secondModelID int64 = 2
	firstModel := modelKeyEmbedder{model: "embedding-model-a"}

	eventID := createSemanticTestTransitionEventAt(t, ctx, database.WriteQuerier, time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	if _, err := NewIndexer(database.WriteQuerier, database.ReadQuerier, firstModel, firstModelID).IndexTransitionEvent(ctx, eventID); err != nil {
		t.Fatalf("index first model: %v", err)
	}

	ids, err := database.ReadQuerier.ListMissingSemanticEventDocumentIDs(ctx, readqueries.ListMissingSemanticEventDocumentIDsParams{
		EmbeddingModelID: sql.NullInt64{Int64: secondModelID, Valid: true},
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("list missing for second model: %v", err)
	}
	if !reflect.DeepEqual(ids, []int64{eventID}) {
		t.Fatalf("expected event to be missing for second model, got %v", ids)
	}

	enqueuer := &recordingBackfillEnqueuer{}
	result, err := NewBackfiller(
		database.ReadQuerier,
		enqueuer,
		secondModelID,
	).BackfillMissing(ctx, 10)
	if err != nil {
		t.Fatalf("backfill second model: %v", err)
	}
	if result != (BackfillResult{Checked: 1, Enqueued: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	if !reflect.DeepEqual(enqueuer.ids, []int64{eventID}) {
		t.Fatalf("expected enqueued ids %v, got %v", []int64{eventID}, enqueuer.ids)
	}
}

func TestBackfillerHasMissing(t *testing.T) {
	ctx := context.Background()

	hasMissing, err := NewBackfiller(staticMissingTransitionEventLister{ids: []int64{42}}, nil, 1).HasMissing(ctx)
	if err != nil {
		t.Fatalf("has missing: %v", err)
	}
	if !hasMissing {
		t.Fatal("expected missing transition event documents")
	}

	hasMissing, err = NewBackfiller(staticMissingTransitionEventLister{}, nil, 1).HasMissing(ctx)
	if err != nil {
		t.Fatalf("has missing empty: %v", err)
	}
	if hasMissing {
		t.Fatal("expected no missing transition event documents")
	}
}

func TestBackfillerContinuesAfterEnqueueFailure(t *testing.T) {
	ctx := context.Background()
	enqueuer := &recordingBackfillEnqueuer{failID: 2}
	result, err := NewBackfiller(staticMissingTransitionEventLister{ids: []int64{1, 2, 3}}, enqueuer, 1).BackfillMissing(ctx, 10)
	if err == nil {
		t.Fatal("expected partial backfill error")
	}

	if result != (BackfillResult{Checked: 3, Enqueued: 2, Failed: 1}) {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	wantEnqueuedIDs := []int64{1, 2, 3}
	if !reflect.DeepEqual(enqueuer.ids, wantEnqueuedIDs) {
		t.Fatalf("expected attempted ids %v, got %v", wantEnqueuedIDs, enqueuer.ids)
	}
}

func TestBackfillCoordinatorStoresPartialFailure(t *testing.T) {
	ctx := context.Background()
	coordinator := NewBackfillCoordinator(
		ctx,
		NewBackfiller(
			staticMissingTransitionEventLister{ids: []int64{1, 2, 3}},
			&recordingBackfillEnqueuer{failID: 2},
			1,
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
			if status.LastResult != (BackfillResult{Checked: 3, Enqueued: 2, Failed: 1}) {
				t.Fatalf("unexpected final status: %+v", status)
			}
			if status.LastError != "Semantic backfill failed. Check sidecar logs." {
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
	enqueuer := &blockingBackfillEnqueuer{
		started: started,
		release: release,
	}
	coordinator := NewBackfillCoordinator(
		ctx,
		NewBackfiller(staticMissingTransitionEventLister{ids: []int64{1}}, enqueuer, 1),
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
			if status.LastResult != (BackfillResult{Checked: 1, Enqueued: 1}) {
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

func (s staticMissingTransitionEventLister) ListMissingSemanticEventDocumentIDs(context.Context, readqueries.ListMissingSemanticEventDocumentIDsParams) ([]int64, error) {
	return s.ids, s.err
}

type recordingBackfillEnqueuer struct {
	ids    []int64
	failID int64
}

func (r *recordingBackfillEnqueuer) EnqueueTransitionEvent(_ context.Context, transitionEventID int64) error {
	r.ids = append(r.ids, transitionEventID)
	if transitionEventID == r.failID {
		return fmt.Errorf("enqueue failed")
	}
	return nil
}

type blockingBackfillEnqueuer struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (b *blockingBackfillEnqueuer) EnqueueTransitionEvent(ctx context.Context, transitionEventID int64) error {
	close(b.started)
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
