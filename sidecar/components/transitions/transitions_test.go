package transitions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

func TestGetTransitionEventsFiltersByTime(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service := NewService(NewServiceParams{Querier: database.ReadQuerier})

	firstStarted := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	secondStarted := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	thirdStarted := time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC)
	createTransitionEvent(t, ctx, database, "VSCode", "Editor", firstStarted)
	secondID := createTransitionEvent(t, ctx, database, "Google Chrome", "Docs", secondStarted)
	createTransitionEvent(t, ctx, database, "Slack", "Team", thirdStarted)
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
}

func TestSubscribeStreamsNewEventsOnly(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service := NewService(NewServiceParams{Querier: database.ReadQuerier})

	oldID := createTransitionEvent(t, ctx, database, "VSCode", "Editor", time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC))
	subscription := service.Subscribe(SubscribeParams{})
	defer subscription.Close()

	newID := createTransitionEvent(t, ctx, database, "Google Chrome", "Docs", time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	if err := service.PublishTransitionEvent(ctx, PublishTransitionEventParams{ID: newID}); err != nil {
		t.Fatalf("publish transition event: %v", err)
	}

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
	database := newTestDatabase(t, ctx)
	service := NewService(NewServiceParams{Querier: database.ReadQuerier})
	threshold := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	subscription := service.Subscribe(SubscribeParams{Filters: Filters{StartedAt: &threshold}})
	defer subscription.Close()

	filteredID := createTransitionEvent(t, ctx, database, "VSCode", "Editor", threshold.Add(-time.Hour))
	if err := service.PublishTransitionEvent(ctx, PublishTransitionEventParams{ID: filteredID}); err != nil {
		t.Fatalf("publish filtered transition event: %v", err)
	}

	matchingID := createTransitionEvent(t, ctx, database, "Google Chrome", "Docs", threshold)
	if err := service.PublishTransitionEvent(ctx, PublishTransitionEventParams{ID: matchingID}); err != nil {
		t.Fatalf("publish matching transition event: %v", err)
	}

	select {
	case event := <-subscription.Events:
		if event.ID != matchingID {
			t.Fatalf("expected matching event id %d, got %d", matchingID, event.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for matching streamed transition event")
	}
}

func newTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	database, err := db.New(ctx, db.Options{
		Engine:         db.EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
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

func createTransitionEvent(t *testing.T, ctx context.Context, database *db.Database, appName string, tab string, startedAt time.Time) int64 {
	t.Helper()
	reporter := reporter.NewDatabaseReporter(database.WriteConn)
	id, err := reporter.Record(ctx, session.Transition{
		From: &session.Session{
			Key: session.AppKey{
				AppName:  appName,
				TabTitle: tab,
			},
			StartedAt: startedAt,
			EndedAt:   startedAt.Add(5 * time.Minute),
			Duration:  5 * time.Minute,
		},
		Reason: session.ReasonFocusChange,
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}
	if id == 0 {
		t.Fatal("expected persisted transition event id")
	}

	return id
}
