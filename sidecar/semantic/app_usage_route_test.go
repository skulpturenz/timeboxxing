package semantic

import (
	"testing"
	"time"
)

func TestResolveAppUsagePeriod(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 6, 30, 10, 30, 0, 0, loc)
	tests := []struct {
		name      string
		question  string
		wantStart time.Time
		wantEnd   time.Time
		wantLabel string
	}{
		{
			name:      "today",
			question:  "top apps today",
			wantStart: time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantLabel: "Today",
		},
		{
			name:      "yesterday",
			question:  "applications used most yesterday",
			wantStart: time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantLabel: "Yesterday",
		},
		{
			name:      "this week",
			question:  "top apps this week",
			wantStart: time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 6, 0, 0, 0, 0, loc),
			wantLabel: "This week",
		},
		{
			name:      "last week",
			question:  "time by application last week",
			wantStart: time.Date(2026, 6, 22, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantLabel: "Last week",
		},
		{
			name:      "this month",
			question:  "top apps this month",
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantLabel: "This month",
		},
		{
			name:      "last month",
			question:  "top apps last month",
			wantStart: time.Date(2026, 5, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, loc),
			wantLabel: "Last month",
		},
		{
			name:      "explicit date",
			question:  "top apps on 2026-06-29",
			wantStart: time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantLabel: "June 29, 2026",
		},
		{
			name:      "past days",
			question:  "top apps past 7 days",
			wantStart: time.Date(2026, 6, 23, 10, 30, 0, 0, loc),
			wantEnd:   now,
			wantLabel: "Past 7 days",
		},
		{
			name:      "past hours",
			question:  "top apps last 24 hours",
			wantStart: time.Date(2026, 6, 29, 10, 30, 0, 0, loc),
			wantEnd:   now,
			wantLabel: "Past 24 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, gotLabel, ok := resolveAppUsagePeriod(tt.question, now, loc)
			if !ok {
				t.Fatal("expected period to resolve")
			}
			if !gotStart.Equal(tt.wantStart) || !gotEnd.Equal(tt.wantEnd) || gotLabel != tt.wantLabel {
				t.Fatalf("unexpected period start=%s end=%s label=%q", gotStart, gotEnd, gotLabel)
			}
		})
	}
}

func TestResolveAppUsageChartRouteRequiresPeriod(t *testing.T) {
	route := resolveAppUsageChartRoute("Which applications did I use the most?", time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC), time.UTC)

	if !route.Recognized || !route.NeedsPeriod {
		t.Fatalf("expected recognized route requiring period, got %#v", route)
	}
}
