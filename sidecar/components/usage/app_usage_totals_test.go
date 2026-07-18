package usage

import (
	"context"
	"testing"
	"time"
)

func TestGetAppUsageTotalsAggregatesClipsSortsAndLimits(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service, _ := newTestService(t, database)
	windowStart := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(4 * time.Hour)

	createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Google Chrome",
		Browser:         true,
		StartedAt:       windowStart.Add(-30 * time.Minute),
		EndedAt:         windowStart.Add(90 * time.Minute),
	})
	createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Slack",
		StartedAt:       windowStart.Add(2 * time.Hour),
		EndedAt:         windowStart.Add(3 * time.Hour),
	})
	createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "VS Code",
		StartedAt:       windowStart.Add(3 * time.Hour),
		EndedAt:         windowStart.Add(3*time.Hour + 45*time.Minute),
	})

	totals, err := service.GetAppUsageTotals(ctx, AppUsageTotalsParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowEnd},
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("get app usage totals: %v", err)
	}

	if totals.TotalDuration != 3*time.Hour+15*time.Minute {
		t.Fatalf("unexpected total duration %s", totals.TotalDuration)
	}
	if len(totals.Buckets) != 2 {
		t.Fatalf("expected two limited buckets, got %#v", totals.Buckets)
	}
	if totals.Buckets[0].Name != "Google Chrome" || totals.Buckets[0].Source != SourceBrowser || totals.Buckets[0].Duration != 90*time.Minute {
		t.Fatalf("unexpected first bucket %#v", totals.Buckets[0])
	}
	if totals.Buckets[1].Name != "Slack" || totals.Buckets[1].Duration != time.Hour {
		t.Fatalf("unexpected second bucket %#v", totals.Buckets[1])
	}
}

func TestGetAppUsageTotalsIdleRequiresOptIn(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service, _ := newTestService(t, database)
	windowStart := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	window := Window{StartedAt: windowStart, EndedAt: windowStart.Add(2 * time.Hour)}

	createTransitionEvent(t, ctx, database, transitionEventFixture{
		ApplicationName: "Slack",
		StartedAt:       windowStart,
		EndedAt:         windowStart.Add(30 * time.Minute),
	})
	createTransitionEvent(t, ctx, database, transitionEventFixture{
		Idle:      true,
		StartedAt: windowStart.Add(30 * time.Minute),
		EndedAt:   windowStart.Add(90 * time.Minute),
	})

	withoutIdle, err := service.GetAppUsageTotals(ctx, AppUsageTotalsParams{Window: window})
	if err != nil {
		t.Fatalf("get totals without idle: %v", err)
	}
	if withoutIdle.TotalDuration != 30*time.Minute || len(withoutIdle.Buckets) != 1 {
		t.Fatalf("expected only active app totals, got %#v", withoutIdle)
	}

	withIdle, err := service.GetAppUsageTotals(ctx, AppUsageTotalsParams{Window: window, IncludeIdle: true})
	if err != nil {
		t.Fatalf("get totals with idle: %v", err)
	}
	if withIdle.TotalDuration != 90*time.Minute || len(withIdle.Buckets) != 2 {
		t.Fatalf("expected idle to be included, got %#v", withIdle)
	}
	if withIdle.Buckets[0].Name != "Idle" || withIdle.Buckets[0].Source != SourceIdle {
		t.Fatalf("expected idle first by duration, got %#v", withIdle.Buckets)
	}
}

func TestGetAppUsageTotalsIncludesActiveSession(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	windowStart := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	now := windowStart.Add(90 * time.Minute)
	service, _ := newTestService(t, database,
		withClock(func() time.Time { return now }),
	)
	createOpenTimelineEvent(t, ctx, database, openTimelineFixture{
		ApplicationName:       "Linear",
		ApplicationIdentifier: "com.linear",
		ApplicationPath:       "/Applications/Linear.app",
		StartedAt:             windowStart.Add(30 * time.Minute),
	})

	totals, err := service.GetAppUsageTotals(ctx, AppUsageTotalsParams{
		Window: Window{StartedAt: windowStart, EndedAt: windowStart.Add(4 * time.Hour)},
	})
	if err != nil {
		t.Fatalf("get app usage totals: %v", err)
	}

	if totals.TotalDuration != time.Hour {
		t.Fatalf("expected active duration, got %s", totals.TotalDuration)
	}
	if len(totals.Buckets) != 1 || totals.Buckets[0].Name != "Linear" || totals.Buckets[0].ApplicationIdentifier != "com.linear" {
		t.Fatalf("unexpected active bucket %#v", totals.Buckets)
	}
}
