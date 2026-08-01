package ama

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelineModels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func TestAppUsageToolRunnerTimelineFiltersTruncatesAndSanitizesURL(t *testing.T) {
	ctx := context.Background()
	runner, registry := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)

	amaIDs := seedAMAUsage(t, ctx, registry,
		amaUsageEventParams{
			appName:   "Google Chrome",
			tab:       "Docs",
			startedAt: startedAt.Add(-5 * time.Minute),
			endedAt:   startedAt.Add(5 * time.Minute),
			browser:   true,
			cdpURL:    "https://www.example.com/private/path?token=secret",
		},
		amaUsageEventParams{
			appName:   "Slack",
			startedAt: startedAt.Add(10 * time.Minute),
			endedAt:   startedAt.Add(20 * time.Minute),
		},
		amaUsageEventParams{
			startedAt: startedAt.Add(20 * time.Minute),
			endedAt:   startedAt.Add(40 * time.Minute),
			idle:      true,
		},
		amaUsageEventParams{
			appName:   "VSCode",
			startedAt: startedAt.Add(40 * time.Minute),
			endedAt:   startedAt.Add(45 * time.Minute),
		},
	)
	chromeID := amaIDs[0]

	result, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind:   semantic.StructuredQueryKindTimeline,
		Window: semantic.TimeWindow{StartedAt: startedAt, EndedAt: endedAt},
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("execute structured timeline: %v", err)
	}

	timeline := result.Artifacts[0].UsageTimeline
	if timeline == nil {
		t.Fatalf("expected usage timeline artifact, got %#v", result.Artifacts)
	}
	if timeline.TotalEventCount != 3 || !timeline.Truncated || len(timeline.Events) != 2 {
		t.Fatalf("unexpected timeline counts: %#v", timeline)
	}
	if timeline.TotalDurationSeconds != 20*60 {
		t.Fatalf("expected clipped non-idle total duration 1200, got %d", timeline.TotalDurationSeconds)
	}
	first := timeline.Events[0]
	if first.TransitionEventID != chromeID {
		t.Fatalf("expected first transition id %d, got %d", chromeID, first.TransitionEventID)
	}
	if !first.StartedAt.Equal(startedAt) || first.DurationSeconds != 5*60 {
		t.Fatalf("expected first event to be clipped to window start, got %#v", first)
	}
	if first.URLHost != "example.com" {
		t.Fatalf("expected sanitized URL host, got %q", first.URLHost)
	}
	if strings.Contains(result.Content, "token=secret") || strings.Contains(result.Content, "https://") {
		t.Fatalf("expected timeline content to avoid full URLs, got %s", result.Content)
	}
}

func TestAppUsageToolRunnerTimelineCanIncludeIdle(t *testing.T) {
	ctx := context.Background()
	runner, registry := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)
	_ = seedAMAUsage(t, ctx, registry,
		amaUsageEventParams{
			startedAt: startedAt.Add(5 * time.Minute),
			endedAt:   startedAt.Add(25 * time.Minute),
			idle:      true,
		},
	)

	withoutIdle, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind:   semantic.StructuredQueryKindTimeline,
		Window: semantic.TimeWindow{StartedAt: startedAt, EndedAt: endedAt},
	})
	if err != nil {
		t.Fatalf("execute structured timeline without idle: %v", err)
	}
	if withoutIdle.Artifacts[0].UsageTimeline.TotalEventCount != 0 {
		t.Fatalf("expected idle event to be excluded by default, got %#v", withoutIdle.Artifacts[0].UsageTimeline)
	}

	withIdle, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind:        semantic.StructuredQueryKindTimeline,
		Window:      semantic.TimeWindow{StartedAt: startedAt, EndedAt: endedAt},
		IncludeIdle: true,
	})
	if err != nil {
		t.Fatalf("execute structured timeline with idle: %v", err)
	}
	timeline := withIdle.Artifacts[0].UsageTimeline
	if timeline.TotalEventCount != 1 || len(timeline.Events) != 1 || !timeline.Events[0].Idle {
		t.Fatalf("expected idle event when include_idle is true, got %#v", timeline)
	}
}

func TestAppUsageToolRunnerRejectsInvalidStructuredWindows(t *testing.T) {
	ctx := context.Background()
	runner, _ := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)

	_, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind: semantic.StructuredQueryKindTimeline,
		Window: semantic.TimeWindow{
			StartedAt: startedAt,
			EndedAt:   startedAt,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "ended_at must be after started_at") {
		t.Fatalf("expected invalid window error, got %v", err)
	}

	_, err = runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind: "app_totals",
		Window: semantic.TimeWindow{
			StartedAt: startedAt,
			EndedAt:   startedAt.Add(time.Hour),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported structured query kind") {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}

type amaUsageEventParams struct {
	appName   string
	tab       string
	startedAt time.Time
	endedAt   time.Time
	browser   bool
	idle      bool
	cdpURL    string
}

func newAppUsageToolRunnerTestServices(t *testing.T, ctx context.Context) (*AppUsageToolRunner, *services.Services[any, any]) {
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

	registry := services.New()
	db.Register(registry, database)
	runner := NewAppUsageToolRunner(registry)
	runner.location = time.UTC
	return runner, registry
}

// seedAMAUsage writes a sequence of stretches through the real timeline ingest. The event store
// chains its entries — each observation closes the entry the previous one opened — so a gap between
// two stretches is not representable and is filled with idle, which is what it actually was. It
// returns the timeline entry id backing each stretch, in the order given.
func seedAMAUsage(
	t *testing.T,
	ctx context.Context,
	registry *services.Services[any, any],
	stretches ...amaUsageEventParams,
) []int64 {
	t.Helper()

	observations := []timelineModels.ForegroundProcess{}
	for i, stretch := range stretches {
		if i > 0 {
			if previous := stretches[i-1]; previous.endedAt.Before(stretch.startedAt) {
				observations = append(observations, amaIdleObservation(previous.endedAt))
			}
		}

		observations = append(observations, amaObservation(stretch))
	}
	if len(stretches) > 0 {
		// close the final stretch; nothing was observed after it
		observations = append(observations, amaIdleObservation(stretches[len(stretches)-1].endedAt))
	}

	var previous *timelineModels.ForegroundProcess
	for i := range observations {
		cmd := componentTimeline.CommandUpsertForegroundProcess{
			PreviousProcess: previous,
			ActiveProcess:   observations[i],
		}
		if err := cmd.Exec(ctx, registry); err != nil {
			t.Fatalf("seed observation %d: %v", i, err)
		}

		previous = &observations[i]
	}

	byStart := amaTimelineIDsByStart(t, ctx, registry)
	ids := make([]int64, 0, len(stretches))
	for _, stretch := range stretches {
		id, ok := byStart[stretch.startedAt.UTC()]
		if !ok {
			t.Fatalf("no timeline entry starting at %v", stretch.startedAt)
		}

		ids = append(ids, id)
	}

	return ids
}

func amaObservation(params amaUsageEventParams) timelineModels.ForegroundProcess {
	if params.idle {
		return amaIdleObservation(params.startedAt)
	}

	name := params.appName
	identifier := "test." + strings.ReplaceAll(name, " ", ".")
	path := "/Applications/" + name + ".app"
	// distinct applications must get distinct pids: ForegroundProcess.IsEqual keys on pid, so
	// sharing one would make consecutive observations of different applications look like the
	// same one still being in focus
	pid := amaPID(name)

	observation := timelineModels.ForegroundProcess{
		AppName:       &name,
		AppIdentifier: &identifier,
		AppPath:       &path,
		PID:           &pid,
		Timestamp:     params.startedAt,
	}
	if params.browser {
		observation.Enrichments.Browser = timelineModels.Browser{
			Vendor: name,
			Tab:    params.tab,
			CdpURL: params.cdpURL,
		}
	}

	return observation
}

// amaPID derives a stable, distinct pid per application name.
func amaPID(name string) int64 {
	pid := int64(4242)
	for _, r := range name {
		pid = pid*31 + int64(r)
	}

	return pid & 0xffff
}

func amaIdleObservation(at time.Time) timelineModels.ForegroundProcess {
	return timelineModels.ForegroundProcess{Timestamp: at, Idle: true}
}

// amaTimelineIDsByStart maps each entry to the instant its opening observation was recorded, which
// is the only stable way to name an entry the ingest created implicitly.
func amaTimelineIDsByStart(t *testing.T, ctx context.Context, registry *services.Services[any, any]) map[time.Time]int64 {
	t.Helper()

	database, ok := db.FromServices(registry)
	if !ok {
		t.Fatal("database is not registered")
	}

	query := `SELECT timeline.id, initial.created_at_utc
		FROM timeline
		JOIN foreground_processes initial ON initial.id = timeline.initial_foreground_process_id`
	rows, err := database.ReadConn.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("read timeline entries: %v", err)
	}
	defer rows.Close()

	byStart := map[time.Time]int64{}
	for rows.Next() {
		var id int64
		var startedAt time.Time
		if err := rows.Scan(&id, &startedAt); err != nil {
			t.Fatalf("scan timeline entry: %v", err)
		}

		byStart[startedAt.UTC()] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate timeline entries: %v", err)
	}

	return byStart
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
