package usage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func TestGetEventsReturnsOverlappingUsage(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service, _ := newTestService(t, database)
	windowStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)

	beforeID := createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Before",
		StartedAt:       windowStart.Add(-2 * time.Hour),
		EndedAt:         windowStart.Add(-time.Hour),
	})
	crossingID := createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Google Chrome",
		Browser:         true,
		Tab:             stringPtr("Docs"),
		StartedAt:       windowStart.Add(-15 * time.Minute),
		EndedAt:         windowStart.Add(15 * time.Minute),
	})
	insideID := createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Slack",
		StartedAt:       windowStart.Add(time.Hour),
		EndedAt:         windowStart.Add(2 * time.Hour),
	})

	events, err := service.GetEvents(ctx, GetEventsParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowEnd},
	})
	if err != nil {
		t.Fatalf("get usage events: %v", err)
	}

	gotIDs := []int64{}
	for _, event := range events {
		gotIDs = append(gotIDs, event.ID)
	}
	if containsID(gotIDs, beforeID) {
		t.Fatalf("expected event before window to be filtered, got ids %v", gotIDs)
	}
	if !containsID(gotIDs, crossingID) || !containsID(gotIDs, insideID) {
		t.Fatalf("expected overlapping events %d and %d, got ids %v", crossingID, insideID, gotIDs)
	}
}

func TestUsageEventMapping(t *testing.T) {
	startedAt := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)

	browser := eventFromTransition(componentTransitions.Event{
		ID:                    10,
		ApplicationName:       "Google Chrome",
		ApplicationIdentifier: "com.google.Chrome",
		ApplicationPath:       "/Applications/Google Chrome.app",
		Tab:                   "Client dashboard",
		Browser:               true,
		StartedAt:             startedAt,
		EndedAt:               startedAt.Add(time.Minute),
		Reason:                "tab_change",
	})
	if browser.Source != SourceBrowser || browser.Title != "Client dashboard" || browser.SourceName != "Google Chrome" {
		t.Fatalf("unexpected browser mapping: %#v", browser)
	}
	if browser.ApplicationIdentifier != "com.google.Chrome" || browser.ApplicationPath != "/Applications/Google Chrome.app" {
		t.Fatalf("expected browser identity to be preserved, got %#v", browser)
	}

	idle := eventFromTransition(componentTransitions.Event{
		ID:                    11,
		ApplicationName:       "Google Chrome",
		ApplicationIdentifier: "com.google.Chrome",
		ApplicationPath:       "/Applications/Google Chrome.app",
		Idle:                  true,
		StartedAt:             startedAt,
		EndedAt:               startedAt.Add(time.Minute),
		Reason:                "idle",
	})
	if idle.Source != SourceIdle || idle.Title != "Idle" || idle.SourceName != "Idle" {
		t.Fatalf("unexpected idle mapping: %#v", idle)
	}
	if idle.ApplicationIdentifier != "" || idle.ApplicationPath != "" {
		t.Fatalf("expected idle identity to be empty, got %#v", idle)
	}
}

func TestGetEventsIncludesActiveSession(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	windowStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	now := windowStart.Add(10 * time.Hour)
	activeStartedAt := now.Add(-20 * time.Minute)
	service, _ := newTestService(t, database,
		withActiveSessions(fakeActiveSessionProvider{
			current: &session.Session{
				Key: session.AppKey{
					AppName:  "Google Chrome",
					TabTitle: "Client dashboard",
					CDPURL:   "http://localhost:9222/json",
				},
				ApplicationIdentity: session.AppIdentity{
					Identifier: "com.google.Chrome",
					Path:       "/Applications/Google Chrome.app",
				},
				StartedAt: activeStartedAt,
			},
		}),
		withClock(func() time.Time { return now }),
	)

	events, err := service.GetEvents(ctx, GetEventsParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowStart.Add(24 * time.Hour)},
	})
	if err != nil {
		t.Fatalf("get usage events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected active event only, got %#v", events)
	}
	event := events[0]
	if event.ID != -1 || !event.Active || event.Reason != "active" {
		t.Fatalf("expected synthetic active event, got %#v", event)
	}
	if event.Source != SourceBrowser || event.Title != "Client dashboard" || event.SourceName != "Google Chrome" {
		t.Fatalf("unexpected active browser mapping: %#v", event)
	}
	if !event.StartedAt.Equal(activeStartedAt) || !event.EndedAt.Equal(now) {
		t.Fatalf("unexpected active event time range: %#v", event)
	}
	if event.ApplicationIdentifier != "com.google.Chrome" || event.ApplicationPath != "/Applications/Google Chrome.app" {
		t.Fatalf("expected active identity to be preserved, got %#v", event)
	}
}

func TestGetEventsExcludesActiveSessionOutsideWindow(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	windowStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	service, _ := newTestService(t, database,
		withActiveSessions(fakeActiveSessionProvider{
			current: &session.Session{
				Key:       session.AppKey{AppName: "Slack"},
				StartedAt: windowStart.Add(25 * time.Hour),
			},
		}),
		withClock(func() time.Time { return windowStart.Add(25 * time.Hour) }),
	)

	events, err := service.GetEvents(ctx, GetEventsParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowStart.Add(24 * time.Hour)},
	})
	if err != nil {
		t.Fatalf("get usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no active events outside window, got %#v", events)
	}
}

func TestSubscribeEmitsInitialAndPeriodicActiveSnapshots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	database := newTestDatabase(t, ctx)
	windowStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	service, _ := newTestService(t, database,
		withActiveSessions(fakeActiveSessionProvider{
			current: &session.Session{
				Key:       session.AppKey{AppName: "Slack"},
				StartedAt: windowStart.Add(9 * time.Hour),
			},
		}),
		withClock(func() time.Time { return windowStart.Add(10 * time.Hour) }),
		withActiveSnapshotInterval(5*time.Millisecond),
	)

	subscription := service.Subscribe(ctx, SubscribeParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowStart.Add(24 * time.Hour)},
	})
	defer subscription.Close()

	first := readUsageEvent(t, subscription.Events)
	second := readUsageEvent(t, subscription.Events)
	if !first.Active || !second.Active {
		t.Fatalf("expected active snapshots, got %#v and %#v", first, second)
	}
}

func TestSubscribeEmitsCompletedTransitionThenActiveSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	database := newTestDatabase(t, ctx)
	windowStart := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	service, transitions := newTestService(t, database,
		withActiveSessions(fakeActiveSessionProvider{
			current: &session.Session{
				Key:       session.AppKey{AppName: "Slack"},
				StartedAt: windowStart.Add(10 * time.Hour),
			},
		}),
		withClock(func() time.Time { return windowStart.Add(10*time.Hour + time.Minute) }),
		withActiveSnapshotInterval(time.Hour),
	)
	subscription := service.Subscribe(ctx, SubscribeParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowStart.Add(24 * time.Hour)},
	})
	defer subscription.Close()
	initial := readUsageEvent(t, subscription.Events)
	if !initial.Active {
		t.Fatalf("expected initial active event, got %#v", initial)
	}

	completedID := createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Google Chrome",
		Browser:         true,
		Tab:             stringPtr("Docs"),
		StartedAt:       windowStart.Add(9 * time.Hour),
		EndedAt:         windowStart.Add(10 * time.Hour),
	})
	if err := transitions.PublishTransitionEvent(ctx, componentTransitions.PublishTransitionEventParams{ID: completedID}); err != nil {
		t.Fatalf("publish transition event: %v", err)
	}

	completed := readUsageEvent(t, subscription.Events)
	active := readUsageEvent(t, subscription.Events)
	if completed.ID != completedID || completed.Active {
		t.Fatalf("expected completed event %d, got %#v", completedID, completed)
	}
	if !active.Active || active.SourceName != "Slack" {
		t.Fatalf("expected active snapshot after completed event, got %#v", active)
	}
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

type testServiceOption func(*services.Services[any, any])

func newTestService(t *testing.T, database *db.Database, opts ...testServiceOption) (*Service, *componentTransitions.Service) {
	t.Helper()
	registry := services.New()
	db.Register(registry, database)
	for _, opt := range opts {
		opt(registry)
	}
	transitions := componentTransitions.NewService(registry)
	return NewService(registry), transitions
}

func withActiveSessions(provider ActiveSessionProvider) testServiceOption {
	return func(registry *services.Services[any, any]) {
		RegisterActiveSessions(registry, provider)
	}
}

func withClock(clock func() time.Time) testServiceOption {
	return func(registry *services.Services[any, any]) {
		RegisterClock(registry, clock)
	}
}

func withActiveSnapshotInterval(interval time.Duration) testServiceOption {
	return func(registry *services.Services[any, any]) {
		RegisterActiveSnapshotInterval(registry, interval)
	}
}

type transitionEventFixture struct {
	ApplicationName       string
	ApplicationIdentifier string
	ApplicationPath       string
	Browser               bool
	Tab                   *string
	Idle                  bool
	StartedAt             time.Time
	EndedAt               time.Time
}

func createTransitionEvent(t *testing.T, ctx context.Context, database *db.Database, fixture transitionEventFixture) int64 {
	t.Helper()
	applicationID := sql.NullInt64{}
	if !fixture.Idle && fixture.ApplicationName != "" {
		id, err := database.WriteQuerier.UpsertApplication(ctx, writequeries.UpsertApplicationParams{
			Name:               fixture.ApplicationName,
			PlatformIdentifier: nullableString(fixture.ApplicationIdentifier),
			Path:               nullableString(fixture.ApplicationPath),
		})
		if err != nil {
			t.Fatalf("upsert application: %v", err)
		}
		applicationID = sql.NullInt64{Int64: id, Valid: true}
	}

	id, err := database.WriteQuerier.CreateTransitionEvent(ctx, writequeries.CreateTransitionEventParams{
		ApplicationID: applicationID,
		Reason:        "focus_change",
		StartedAt:     fixture.StartedAt,
		EndedAt:       fixture.EndedAt,
	})
	if err != nil {
		t.Fatalf("create transition event: %v", err)
	}
	if err := database.WriteQuerier.CreateTransitionEventMetadata(ctx, writequeries.CreateTransitionEventMetadataParams{
		TransitionEventID: id,
		Browser:           fixture.Browser,
		Tab:               fixture.Tab,
		Idle:              fixture.Idle,
	}); err != nil {
		t.Fatalf("create transition event metadata: %v", err)
	}

	return id
}

func stringPtr(value string) *string {
	return &value
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func containsID(ids []int64, id int64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

type fakeActiveSessionProvider struct {
	current *session.Session
}

func (f fakeActiveSessionProvider) CurrentSession() *session.Session {
	if f.current == nil {
		return nil
	}
	copy := *f.current
	return &copy
}

func readUsageEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("usage event channel closed")
		}
		return event
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for usage event")
	}
	return Event{}
}
