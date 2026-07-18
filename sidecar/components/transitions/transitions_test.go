package transitions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/enrichment/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func TestGetTransitionEventsFiltersByTime(t *testing.T) {
	ctx := context.Background()
	service, _ := newTestService(t, ctx)

	firstStarted := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	secondStarted := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	thirdStarted := time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC)
	createTransitionEvent(t, ctx, service, "VSCode", "Editor", firstStarted)
	secondID := createTransitionEvent(t, ctx, service, "Google Chrome", "Docs", secondStarted)
	createTransitionEvent(t, ctx, service, "Slack", "Team", thirdStarted)
	endedAt := secondStarted.Add(5 * time.Minute)

	events, err := service.GetTransitionEvents(ctx, GetTransitionEventsParams{
		Filters: Filters{
			StartedAt: &secondStarted,
			EndedAt:   &endedAt,
		},
	})
	if err != nil {
		t.Fatalf("get transition events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected one transition event, got %d", len(events))
	}
	if events[0].ID != secondID {
		t.Fatalf("expected transition event id %d, got %d", secondID, events[0].ID)
	}
	if events[0].ApplicationName != "Google Chrome" || events[0].Tab != "Docs" || !events[0].Browser {
		t.Fatalf("unexpected mapped transition event: %#v", events[0])
	}
	// platform_identifier was dropped from the schema; only the application path is surfaced now.
	if events[0].ApplicationPath == "" {
		t.Fatalf("expected application path to be mapped, got %#v", events[0])
	}
	if events[0].PID != 4242 {
		t.Fatalf("expected pid 4242, got %d", events[0].PID)
	}
}

func TestRecordTransitionEventRequiresTimestamps(t *testing.T) {
	ctx := context.Background()
	service, _ := newTestService(t, ctx)

	// The event store no longer fills timestamp defaults; started_at and ended_at are required and
	// map to the foreground_processes boundary rows.
	if _, err := service.RecordTransitionEvent(ctx, RecordTransitionEventParams{
		ApplicationName: "VSCode",
		Reason:          "focus_change",
	}); err == nil {
		t.Fatal("expected error when started_at and ended_at are missing")
	}

	events, err := service.GetTransitionEvents(ctx, GetTransitionEventsParams{})
	if err != nil {
		t.Fatalf("get transition events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no persisted transition events, got %d", len(events))
	}
}

func TestSubscribeStreamsNewEventsOnly(t *testing.T) {
	ctx := context.Background()
	service, _ := newTestService(t, ctx)

	oldID := createTransitionEvent(t, ctx, service, "VSCode", "Editor", time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	subscription := service.Subscribe(ctx, SubscribeParams{})
	defer subscription.Close()

	newID := createTransitionEvent(t, ctx, service, "Google Chrome", "Docs", time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))

	select {
	case event := <-subscription.Events:
		if event.ID != newID {
			t.Fatalf("expected streamed event id %d, got %d", newID, event.ID)
		}
		if event.ID == oldID {
			t.Fatalf("stream replayed old event id %d", oldID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for streamed transition event")
	}
}

func TestSubscribeAppliesFilters(t *testing.T) {
	ctx := context.Background()
	service, _ := newTestService(t, ctx)
	threshold := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	subscription := service.Subscribe(ctx, SubscribeParams{Filters: Filters{StartedAt: &threshold}})
	defer subscription.Close()

	createTransitionEvent(t, ctx, service, "VSCode", "Editor", threshold.Add(-time.Hour))

	matchingID := createTransitionEvent(t, ctx, service, "Google Chrome", "Docs", threshold)

	select {
	case event := <-subscription.Events:
		if event.ID != matchingID {
			t.Fatalf("expected matching event id %d, got %d", matchingID, event.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for matching streamed transition event")
	}
}

func newTestService(t *testing.T, ctx context.Context) (*Service, *db.Database) {
	t.Helper()
	database := newTestDatabase(t, ctx)
	registry := services.New()
	db.Register(registry, database)
	return NewService(registry), database
}

func newTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	database, err := db.New(ctx, db.Options{
		DSN: db.NewDSN(filepath.Join(t.TempDir(), "test.db")),
	})
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	return database
}

func createTransitionEvent(t *testing.T, ctx context.Context, service *Service, appName string, tab string, startedAt time.Time) int64 {
	t.Helper()
	id, err := service.RecordTransitionEvent(ctx, RecordTransitionEventParams{
		ApplicationName:       appName,
		ApplicationIdentifier: "test." + appName,
		ApplicationPath:       "/Applications/" + appName + ".app",
		PID:                   4242,
		Reason:                "focus_change",
		StartedAt:             startedAt,
		EndedAt:               startedAt.Add(5 * time.Minute),
		Browser:               browser.IsBrowser(appName) != browser.BrowserNone,
		Tab:                   &tab,
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}
	if id == 0 {
		t.Fatal("expected persisted transition event id")
	}

	return id
}
