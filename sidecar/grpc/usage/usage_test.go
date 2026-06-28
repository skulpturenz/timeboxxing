package usage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetUsageEventsMapsComponentEvents(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	transitions := componentTransitions.NewService(componentTransitions.NewServiceParams{Querier: database.ReadQuerier})
	server := NewServer(NewServerParams{
		Usage: componentUsage.NewService(componentUsage.NewServiceParams{Transitions: transitions}),
	})
	startedAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	id := createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName:       "Google Chrome",
		ApplicationIdentifier: "com.google.Chrome",
		ApplicationPath:       "/Applications/Google Chrome.app",
		Browser:               true,
		Tab:                   stringPtr("Docs"),
		StartedAt:             startedAt,
		EndedAt:               startedAt.Add(5 * time.Minute),
	})

	resp, err := server.GetUsageEvents(ctx, &usagev1.GetUsageEventsRequest{
		StartedAt: timestamppb.New(startedAt.Add(-time.Hour)),
		EndedAt:   timestamppb.New(startedAt.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("get usage events: %v", err)
	}
	events := resp.GetUsageEvents()
	if len(events) != 1 {
		t.Fatalf("expected one usage event, got %d", len(events))
	}
	if events[0].GetId() != id ||
		events[0].GetTitle() != "Docs" ||
		events[0].GetSourceName() != "Google Chrome" ||
		events[0].GetSource() != usagev1.UsageSource_USAGE_SOURCE_BROWSER ||
		events[0].GetApplicationIdentifier() != "com.google.Chrome" ||
		events[0].GetApplicationPath() != "/Applications/Google Chrome.app" {
		t.Fatalf("unexpected usage event: %#v", events[0])
	}
}

func TestGetUsageEventsRejectsInvalidWindows(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	transitions := componentTransitions.NewService(componentTransitions.NewServiceParams{Querier: database.ReadQuerier})
	server := NewServer(NewServerParams{
		Usage: componentUsage.NewService(componentUsage.NewServiceParams{Transitions: transitions}),
	})
	now := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)

	_, err := server.GetUsageEvents(ctx, &usagev1.GetUsageEventsRequest{
		StartedAt: timestamppb.New(now),
		EndedAt:   timestamppb.New(now),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
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
		id, err := database.WriteQuerier.UpsertApplication(ctx, queries.UpsertApplicationParams{
			Name:               fixture.ApplicationName,
			PlatformIdentifier: nullableString(fixture.ApplicationIdentifier),
			Path:               nullableString(fixture.ApplicationPath),
		})
		if err != nil {
			t.Fatalf("upsert application: %v", err)
		}
		applicationID = sql.NullInt64{Int64: id, Valid: true}
	}

	id, err := database.WriteQuerier.CreateTransitionEvent(ctx, queries.CreateTransitionEventParams{
		ApplicationID: applicationID,
		Reason:        "focus_change",
		StartedAt:     fixture.StartedAt,
		EndedAt:       fixture.EndedAt,
	})
	if err != nil {
		t.Fatalf("create transition event: %v", err)
	}
	if err := database.WriteQuerier.CreateTransitionEventMetadata(ctx, queries.CreateTransitionEventMetadataParams{
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
