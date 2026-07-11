package transitions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
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
	if events[0].ApplicationIdentifier == "" || events[0].ApplicationPath == "" {
		t.Fatalf("expected application identity to be mapped, got %#v", events[0])
	}
	if events[0].PID != 4242 {
		t.Fatalf("expected pid 4242, got %d", events[0].PID)
	}
}

func TestRecordTransitionEventCanUseDatabaseTimestampDefaults(t *testing.T) {
	ctx := context.Background()
	service, _ := newTestService(t, ctx)
	before := time.Now().UTC().Add(-time.Second)

	id, err := service.RecordTransitionEvent(ctx, RecordTransitionEventParams{
		ApplicationName: "VSCode",
		Reason:          "focus_change",
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	events, err := service.GetTransitionEvents(ctx, GetTransitionEventsParams{})
	if err != nil {
		t.Fatalf("get transition events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one transition event, got %d", len(events))
	}
	if events[0].ID != id {
		t.Fatalf("expected transition event id %d, got %d", id, events[0].ID)
	}
	if events[0].StartedAt.IsZero() || events[0].EndedAt.IsZero() {
		t.Fatalf("expected timestamp defaults, got %#v", events[0])
	}
	if events[0].StartedAt.Location() != time.UTC || events[0].EndedAt.Location() != time.UTC {
		t.Fatalf("expected UTC timestamps, got %s and %s", events[0].StartedAt.Location(), events[0].EndedAt.Location())
	}
	if events[0].StartedAt.Before(before) || events[0].StartedAt.After(after) {
		t.Fatalf("started_at %s outside expected range %s..%s", events[0].StartedAt, before, after)
	}
	if events[0].EndedAt.Before(before) || events[0].EndedAt.After(after) {
		t.Fatalf("ended_at %s outside expected range %s..%s", events[0].EndedAt, before, after)
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
		Engine: db.EngineSqlite,
		DSN:    filepath.Join(t.TempDir(), "test.db"),
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
