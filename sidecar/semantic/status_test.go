package semantic

import (
	"context"
	"testing"
	"time"
)

func TestIndexStatusServiceStates(t *testing.T) {
	ctx := context.Background()
	database := newSemanticTestDatabase(t, ctx)
	service := NewIndexStatusService(database.ReadQuerier, nil)

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
