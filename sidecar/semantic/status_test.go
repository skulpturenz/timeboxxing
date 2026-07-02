package semantic

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestIndexStatusServiceStates(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	service := NewIndexStatusService(database.ReadQuerier, nil, fakeEmbedder{}.Model())

	status, err := service.Status(ctx)
	if err != nil {
		t.Fatalf("empty status: %v", err)
	}
	if status.State != IndexStateEmpty {
		t.Fatalf("expected empty status, got %+v", status)
	}

	eventID := createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	status, err = service.Status(ctx)
	if err != nil {
		t.Fatalf("indexing status: %v", err)
	}
	if status.State != IndexStateIndexing || status.PendingEventCount != 1 {
		t.Fatalf("expected indexing status with one pending event, got %+v", status)
	}

	if _, err := NewIndexer(database.WriteConn, database.ReadQuerier, fakeEmbedder{}).IndexTransitionEvent(ctx, eventID); err != nil {
		t.Fatalf("index event: %v", err)
	}
	status, err = service.Status(ctx)
	if err != nil {
		t.Fatalf("ready status: %v", err)
	}
	if status.State != IndexStateReady || status.PendingEventCount != 0 || status.IndexedEventCount != 1 {
		t.Fatalf("expected ready status, got %+v", status)
	}
}

func TestIndexStatusServiceExposesBackfillFailure(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	createSemanticTestTransitionEventAt(t, ctx, database.WriteConn, time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	coordinator := NewBackfillCoordinator(
		ctx,
		NewBackfiller(
			staticMissingTransitionEventLister{ids: []int64{1}},
			&recordingBackfillEnqueuer{failID: 1},
			fakeEmbedder{}.Model(),
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if !coordinator.Start(10) {
		t.Fatal("expected backfill to start")
	}
	deadline := time.Now().Add(time.Second)
	for coordinator.Status().Running {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for backfill completion: %+v", coordinator.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}

	status, err := NewIndexStatusService(database.ReadQuerier, coordinator, fakeEmbedder{}.Model()).Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != IndexStateUnavailable {
		t.Fatalf("expected unavailable status, got %+v", status)
	}
	if status.Message != "Semantic backfill failed. Check sidecar logs." {
		t.Fatalf("unexpected status message %q", status.Message)
	}
}
