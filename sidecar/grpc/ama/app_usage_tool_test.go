package ama

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func TestAppUsageToolRunnerTimelineFiltersTruncatesAndSanitizesURL(t *testing.T) {
	ctx := context.Background()
	runner, transitions := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)

	chromeID := recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Google Chrome",
		tab:       "Docs",
		startedAt: startedAt.Add(-5 * time.Minute),
		endedAt:   startedAt.Add(5 * time.Minute),
		browser:   true,
		cdpURL:    "https://www.example.com/private/path?token=secret",
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Slack",
		startedAt: startedAt.Add(10 * time.Minute),
		endedAt:   startedAt.Add(20 * time.Minute),
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		startedAt: startedAt.Add(20 * time.Minute),
		endedAt:   startedAt.Add(40 * time.Minute),
		idle:      true,
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "VSCode",
		startedAt: startedAt.Add(40 * time.Minute),
		endedAt:   startedAt.Add(45 * time.Minute),
	})

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
	runner, transitions := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		startedAt: startedAt.Add(5 * time.Minute),
		endedAt:   startedAt.Add(25 * time.Minute),
		idle:      true,
	})

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

func TestAppUsageToolRunnerHabitSummaryMetrics(t *testing.T) {
	ctx := context.Background()
	runner, transitions := newAppUsageToolRunnerTestServices(t, ctx)
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(6 * time.Hour)

	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Google Chrome",
		startedAt: startedAt,
		endedAt:   startedAt.Add(15 * time.Minute),
		browser:   true,
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Slack",
		startedAt: startedAt.Add(15 * time.Minute),
		endedAt:   startedAt.Add(45 * time.Minute),
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Google Chrome",
		startedAt: startedAt.Add(4 * time.Hour),
		endedAt:   startedAt.Add(4*time.Hour + 15*time.Minute),
		browser:   true,
	})

	result, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind:   semantic.StructuredQueryKindHabits,
		Window: semantic.TimeWindow{StartedAt: startedAt, EndedAt: endedAt},
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("execute structured habits: %v", err)
	}

	summary := result.Artifacts[0].UsageHabitSummary
	if summary == nil {
		t.Fatalf("expected habit summary artifact, got %#v", result.Artifacts)
	}
	if summary.TotalDurationSeconds != 60*60 || summary.SessionCount != 3 || summary.ContextSwitchCount != 2 {
		t.Fatalf("unexpected summary metrics: %#v", summary)
	}
	if summary.AverageSessionSeconds != 20*60 {
		t.Fatalf("expected average session 1200 seconds, got %d", summary.AverageSessionSeconds)
	}
	if summary.LongestSession == nil || summary.LongestSession.SourceName != "Slack" || summary.LongestSession.DurationSeconds != 30*60 {
		t.Fatalf("unexpected longest session: %#v", summary.LongestSession)
	}
	if len(summary.TopSources) == 0 || summary.TopSources[0].Name != "Google Chrome" || summary.TopSources[0].DurationSeconds != 30*60 {
		t.Fatalf("unexpected top sources: %#v", summary.TopSources)
	}
	assertTimeBucket(t, summary.TimeBuckets, "Morning", 45*60, 2)
	assertTimeBucket(t, summary.TimeBuckets, "Afternoon", 15*60, 1)
}

func TestAppUsageToolRunnerComparisonDeltas(t *testing.T) {
	ctx := context.Background()
	runner, transitions := newAppUsageToolRunnerTestServices(t, ctx)
	baselineStart := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	currentStart := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)

	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Google Chrome",
		startedAt: baselineStart,
		endedAt:   baselineStart.Add(30 * time.Minute),
		browser:   true,
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Slack",
		startedAt: baselineStart.Add(45 * time.Minute),
		endedAt:   baselineStart.Add(55 * time.Minute),
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "Google Chrome",
		startedAt: currentStart,
		endedAt:   currentStart.Add(time.Hour),
		browser:   true,
	})
	recordAMAUsageEvent(t, ctx, transitions, amaUsageEventParams{
		appName:   "VSCode",
		startedAt: currentStart.Add(2 * time.Hour),
		endedAt:   currentStart.Add(2*time.Hour + 15*time.Minute),
	})

	result, err := runner.ExecuteStructured(ctx, semantic.StructuredQuery{
		Kind: semantic.StructuredQueryKindComparePeriods,
		Window: semantic.TimeWindow{
			StartedAt: currentStart,
			EndedAt:   currentStart.Add(24 * time.Hour),
		},
		BaselineWindow: semantic.TimeWindow{
			StartedAt: baselineStart,
			EndedAt:   baselineStart.Add(24 * time.Hour),
		},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("execute structured comparison: %v", err)
	}

	comparison := result.Artifacts[0].UsageComparison
	if comparison == nil {
		t.Fatalf("expected usage comparison artifact, got %#v", result.Artifacts)
	}
	if comparison.CurrentTotalDurationSeconds != 75*60 || comparison.BaselineTotalDurationSeconds != 40*60 {
		t.Fatalf("unexpected comparison totals: %#v", comparison)
	}
	if !strings.Contains(result.Content, `"current_total_duration_seconds":4500`) ||
		!strings.Contains(result.Content, `"baseline_total_duration_seconds":2400`) {
		t.Fatalf("expected comparison content to include totals, got %s", result.Content)
	}
	if comparison.DurationDeltaSeconds != 35*60 || comparison.DurationDeltaPercent != 87.5 {
		t.Fatalf("unexpected comparison delta: %#v", comparison)
	}
	if len(comparison.Buckets) != 1 {
		t.Fatalf("expected comparison limit to apply, got %#v", comparison.Buckets)
	}
	chrome := comparison.Buckets[0]
	if chrome.Name != "Google Chrome" || chrome.CurrentDurationSeconds != 60*60 || chrome.BaselineDurationSeconds != 30*60 || chrome.DeltaDurationSeconds != 30*60 {
		t.Fatalf("unexpected Chrome comparison bucket: %#v", chrome)
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
		Kind: semantic.StructuredQueryKindComparePeriods,
		Window: semantic.TimeWindow{
			StartedAt: startedAt,
			EndedAt:   startedAt.Add(time.Hour),
		},
		BaselineWindow: semantic.TimeWindow{
			StartedAt: startedAt,
			EndedAt:   startedAt,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "baseline ended_at must be after started_at") {
		t.Fatalf("expected invalid baseline window error, got %v", err)
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

func newAppUsageToolRunnerTestServices(t *testing.T, ctx context.Context) (*AppUsageToolRunner, *componentTransitions.Service) {
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

	registry := services.New()
	db.Register(registry, database)
	transitions := componentTransitions.NewService(registry)
	usage := componentUsage.NewService(registry)
	runner := NewAppUsageToolRunner(usage)
	runner.location = time.UTC
	return runner, transitions
}

func recordAMAUsageEvent(
	t *testing.T,
	ctx context.Context,
	service *componentTransitions.Service,
	params amaUsageEventParams,
) int64 {
	t.Helper()
	tab := stringPtr(params.tab)
	cdpURL := stringPtr(params.cdpURL)
	id, err := service.RecordTransitionEvent(ctx, componentTransitions.RecordTransitionEventParams{
		ApplicationName:       params.appName,
		ApplicationIdentifier: "test." + strings.ReplaceAll(params.appName, " ", "."),
		ApplicationPath:       "/Applications/" + params.appName + ".app",
		PID:                   4242,
		Reason:                "focus_change",
		StartedAt:             params.startedAt,
		EndedAt:               params.endedAt,
		Browser:               params.browser,
		Tab:                   tab,
		Idle:                  params.idle,
		CDPURL:                cdpURL,
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}
	return id
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func assertTimeBucket(t *testing.T, buckets []semantic.TimeOfDayBucket, label string, durationSeconds int64, sessionCount int64) {
	t.Helper()
	for _, bucket := range buckets {
		if bucket.Label != label {
			continue
		}
		if bucket.DurationSeconds != durationSeconds || bucket.SessionCount != sessionCount {
			t.Fatalf("unexpected %s bucket: %#v", label, bucket)
		}
		return
	}
	t.Fatalf("missing %s bucket in %#v", label, buckets)
}
