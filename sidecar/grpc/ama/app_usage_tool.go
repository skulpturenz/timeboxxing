package ama

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
)

const (
	appUsageToolName        = "get_app_usage_totals"
	usageTimelineToolName   = "get_usage_timeline"
	habitSummaryToolName    = "get_usage_habit_summary"
	usageComparisonToolName = "compare_usage_windows"

	defaultAppUsageLimit      = 5
	defaultTimelineEventLimit = 50
	defaultComparisonLimit    = 8
	maxAppUsageLimit          = 20
	maxTimelineEventLimit     = 100
)

type AppUsageToolRunner struct {
	usage    *componentUsage.Service
	location *time.Location
}

type appUsageToolArgs struct {
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at"`
	Limit       int    `json:"limit"`
	IncludeIdle bool   `json:"include_idle"`
}

type timelineToolArgs struct {
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at"`
	Limit       int    `json:"limit"`
	IncludeIdle bool   `json:"include_idle"`
}

type habitSummaryToolArgs struct {
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at"`
	Limit       int    `json:"limit"`
	IncludeIdle bool   `json:"include_idle"`
}

type comparisonToolArgs struct {
	CurrentStartedAt  string `json:"current_started_at"`
	CurrentEndedAt    string `json:"current_ended_at"`
	BaselineStartedAt string `json:"baseline_started_at"`
	BaselineEndedAt   string `json:"baseline_ended_at"`
	Limit             int    `json:"limit"`
	IncludeIdle       bool   `json:"include_idle"`
}

type usageToolContent struct {
	StartedAt             string                 `json:"started_at,omitempty"`
	EndedAt               string                 `json:"ended_at,omitempty"`
	TimeZone              string                 `json:"timezone,omitempty"`
	TotalDurationSeconds  int64                  `json:"total_duration_seconds,omitempty"`
	Buckets               []appUsageToolBucket   `json:"buckets,omitempty"`
	Events                []timelineToolEvent    `json:"events,omitempty"`
	SessionCount          int64                  `json:"session_count,omitempty"`
	ContextSwitchCount    int64                  `json:"context_switch_count,omitempty"`
	AverageSessionSeconds int64                  `json:"average_session_seconds,omitempty"`
	LongestSession        *timelineToolEvent     `json:"longest_session,omitempty"`
	TimeBuckets           []timeOfDayToolBucket  `json:"time_buckets,omitempty"`
	CurrentStartedAt      string                 `json:"current_started_at,omitempty"`
	CurrentEndedAt        string                 `json:"current_ended_at,omitempty"`
	BaselineStartedAt     string                 `json:"baseline_started_at,omitempty"`
	BaselineEndedAt       string                 `json:"baseline_ended_at,omitempty"`
	CurrentTotalSeconds   int64                  `json:"current_total_duration_seconds,omitempty"`
	BaselineTotalSeconds  int64                  `json:"baseline_total_duration_seconds,omitempty"`
	DurationDeltaSeconds  int64                  `json:"duration_delta_seconds,omitempty"`
	DurationDeltaPercent  float64                `json:"duration_delta_percent,omitempty"`
	ComparisonBuckets     []comparisonToolBucket `json:"comparison_buckets,omitempty"`
	Defaults              map[string]interface{} `json:"defaults,omitempty"`
}

type appUsageToolBucket struct {
	Name                  string `json:"name"`
	SourceType            string `json:"source_type"`
	DurationSeconds       int64  `json:"duration_seconds"`
	SessionCount          int64  `json:"session_count"`
	ApplicationIdentifier string `json:"application_identifier,omitempty"`
	ApplicationPath       string `json:"application_path,omitempty"`
}

type timelineToolEvent struct {
	TransitionEventID     int64  `json:"transition_event_id"`
	Title                 string `json:"title"`
	SourceName            string `json:"source_name"`
	SourceType            string `json:"source_type"`
	StartedAt             string `json:"started_at"`
	EndedAt               string `json:"ended_at"`
	DurationSeconds       int64  `json:"duration_seconds"`
	ApplicationIdentifier string `json:"application_identifier,omitempty"`
	ApplicationPath       string `json:"application_path,omitempty"`
	URLHost               string `json:"url_host,omitempty"`
	Idle                  bool   `json:"idle,omitempty"`
}

type timeOfDayToolBucket struct {
	Label           string `json:"label"`
	DurationSeconds int64  `json:"duration_seconds"`
	SessionCount    int64  `json:"session_count"`
}

type comparisonToolBucket struct {
	Name                    string `json:"name"`
	SourceType              string `json:"source_type"`
	CurrentDurationSeconds  int64  `json:"current_duration_seconds"`
	BaselineDurationSeconds int64  `json:"baseline_duration_seconds"`
	DeltaDurationSeconds    int64  `json:"delta_duration_seconds"`
	CurrentSessionCount     int64  `json:"current_session_count"`
	BaselineSessionCount    int64  `json:"baseline_session_count"`
}

func NewAppUsageToolRunner(usage *componentUsage.Service) *AppUsageToolRunner {
	return &AppUsageToolRunner{
		usage:    usage,
		location: time.Local,
	}
}

func (r *AppUsageToolRunner) Tools() []semantic.ChatTool {
	return []semantic.ChatTool{
		{
			Name:        appUsageToolName,
			Description: "Returns total captured usage time per application for an exact time window. Browser activity is grouped under the browser application.",
			Parameters:  usageWindowToolParameters(defaultAppUsageLimit),
		},
		{
			Name:        usageTimelineToolName,
			Description: "Returns exact usage transition events for an exact time window, including transition ids, titles, sources, durations, and URL hosts only.",
			Parameters:  usageWindowToolParameters(defaultTimelineEventLimit),
		},
		{
			Name:        habitSummaryToolName,
			Description: "Returns usage habit metrics for an exact time window, including top sources, time-of-day buckets, longest session, average session, and context switches.",
			Parameters:  usageWindowToolParameters(defaultAppUsageLimit),
		},
		{
			Name:        usageComparisonToolName,
			Description: "Compares total usage and source totals between two exact time windows.",
			Parameters:  comparisonToolParameters(),
		},
	}
}

func usageWindowToolParameters(defaultLimit int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"started_at": map[string]any{
				"type":        "string",
				"description": "Inclusive RFC3339 timestamp for the start of the usage window.",
			},
			"ended_at": map[string]any{
				"type":        "string",
				"description": "Exclusive RFC3339 timestamp for the end of the usage window.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Maximum number of rows to return. Default is %d.", defaultLimit),
				"minimum":     1,
			},
			"include_idle": map[string]any{
				"type":        "boolean",
				"description": "Whether idle time should be included. Default false; only true when the user explicitly asks for idle time.",
			},
		},
		"required":             []string{"started_at", "ended_at"},
		"additionalProperties": false,
	}
}

func comparisonToolParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"current_started_at":  map[string]any{"type": "string"},
			"current_ended_at":    map[string]any{"type": "string"},
			"baseline_started_at": map[string]any{"type": "string"},
			"baseline_ended_at":   map[string]any{"type": "string"},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Maximum number of comparison buckets to return. Default is %d.", defaultComparisonLimit),
				"minimum":     1,
			},
			"include_idle": map[string]any{
				"type":        "boolean",
				"description": "Whether idle time should be included. Default false.",
			},
		},
		"required": []string{
			"current_started_at",
			"current_ended_at",
			"baseline_started_at",
			"baseline_ended_at",
		},
		"additionalProperties": false,
	}
}

func (r *AppUsageToolRunner) Execute(ctx context.Context, call semantic.ChatToolCall) (semantic.ToolResult, error) {
	if r == nil || r.usage == nil {
		return semantic.ToolResult{}, fmt.Errorf("usage service is required")
	}

	switch strings.TrimSpace(call.Name) {
	case appUsageToolName:
		var args appUsageToolArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return semantic.ToolResult{}, fmt.Errorf("parse tool arguments: %w", err)
		}
		startedAt, endedAt, err := parseToolWindow(args.StartedAt, args.EndedAt)
		if err != nil {
			return semantic.ToolResult{}, err
		}
		return r.appUsageTotals(ctx, startedAt, endedAt, args.Limit, args.IncludeIdle)

	case usageTimelineToolName:
		var args timelineToolArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return semantic.ToolResult{}, fmt.Errorf("parse tool arguments: %w", err)
		}
		startedAt, endedAt, err := parseToolWindow(args.StartedAt, args.EndedAt)
		if err != nil {
			return semantic.ToolResult{}, err
		}
		return r.usageTimeline(ctx, startedAt, endedAt, args.Limit, args.IncludeIdle)

	case habitSummaryToolName:
		var args habitSummaryToolArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return semantic.ToolResult{}, fmt.Errorf("parse tool arguments: %w", err)
		}
		startedAt, endedAt, err := parseToolWindow(args.StartedAt, args.EndedAt)
		if err != nil {
			return semantic.ToolResult{}, err
		}
		return r.habitSummary(ctx, startedAt, endedAt, args.Limit, args.IncludeIdle)

	case usageComparisonToolName:
		var args comparisonToolArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return semantic.ToolResult{}, fmt.Errorf("parse tool arguments: %w", err)
		}
		currentStart, currentEnd, err := parseToolWindow(args.CurrentStartedAt, args.CurrentEndedAt)
		if err != nil {
			return semantic.ToolResult{}, fmt.Errorf("current window: %w", err)
		}
		baselineStart, baselineEnd, err := parseToolWindow(args.BaselineStartedAt, args.BaselineEndedAt)
		if err != nil {
			return semantic.ToolResult{}, fmt.Errorf("baseline window: %w", err)
		}
		return r.usageComparison(ctx, currentStart, currentEnd, baselineStart, baselineEnd, args.Limit, args.IncludeIdle)

	default:
		return semantic.ToolResult{}, fmt.Errorf("unknown tool %q", call.Name)
	}
}

func (r *AppUsageToolRunner) ExecuteStructured(ctx context.Context, query semantic.StructuredQuery) (semantic.ToolResult, error) {
	if r == nil || r.usage == nil {
		return semantic.ToolResult{}, fmt.Errorf("usage service is required")
	}
	if !query.Window.EndedAt.After(query.Window.StartedAt) {
		return semantic.ToolResult{}, fmt.Errorf("ended_at must be after started_at")
	}

	switch query.Kind {
	case semantic.StructuredQueryKindAppTotals:
		return r.appUsageTotals(ctx, query.Window.StartedAt, query.Window.EndedAt, query.Limit, query.IncludeIdle)
	case semantic.StructuredQueryKindTimeline:
		return r.usageTimeline(ctx, query.Window.StartedAt, query.Window.EndedAt, query.Limit, query.IncludeIdle)
	case semantic.StructuredQueryKindHabits:
		return r.habitSummary(ctx, query.Window.StartedAt, query.Window.EndedAt, query.Limit, query.IncludeIdle)
	case semantic.StructuredQueryKindComparePeriods:
		if !query.BaselineWindow.EndedAt.After(query.BaselineWindow.StartedAt) {
			return semantic.ToolResult{}, fmt.Errorf("baseline ended_at must be after started_at")
		}
		return r.usageComparison(
			ctx,
			query.Window.StartedAt,
			query.Window.EndedAt,
			query.BaselineWindow.StartedAt,
			query.BaselineWindow.EndedAt,
			query.Limit,
			query.IncludeIdle,
		)
	default:
		return semantic.ToolResult{}, fmt.Errorf("unsupported structured query kind %q", query.Kind)
	}
}

func (r *AppUsageToolRunner) appUsageTotals(ctx context.Context, startedAt, endedAt time.Time, limit int, includeIdle bool) (semantic.ToolResult, error) {
	limit = normalizedLimit(limit, defaultAppUsageLimit, maxAppUsageLimit)
	totals, err := r.usage.GetAppUsageTotals(ctx, componentUsage.AppUsageTotalsParams{
		Window: componentUsage.Window{
			StartedAt: startedAt,
			EndedAt:   endedAt,
		},
		Limit:       limit,
		IncludeIdle: includeIdle,
	})
	if err != nil {
		return semantic.ToolResult{}, err
	}

	zone := r.timeZone()
	buckets := semanticBuckets(totals.Buckets)
	contentBuckets := contentBucketsFromSemantic(buckets)
	chart := &semantic.AppUsageChart{
		StartedAt:            startedAt,
		EndedAt:              endedAt,
		TimeZone:             zone,
		TotalDurationSeconds: durationSeconds(totals.TotalDuration),
		Buckets:              buckets,
	}
	content, err := json.Marshal(usageToolContent{
		StartedAt:            startedAt.Format(time.RFC3339),
		EndedAt:              endedAt.Format(time.RFC3339),
		TimeZone:             zone,
		TotalDurationSeconds: chart.TotalDurationSeconds,
		Buckets:              contentBuckets,
		Defaults: map[string]interface{}{
			"limit":        defaultAppUsageLimit,
			"include_idle": false,
		},
	})
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("marshal tool result: %w", err)
	}

	return semantic.ToolResult{
		Content: string(content),
		Artifacts: []semantic.Artifact{{
			Type:          semantic.ArtifactTypeAppUsageChart,
			AppUsageChart: chart,
		}},
	}, nil
}

func (r *AppUsageToolRunner) usageTimeline(ctx context.Context, startedAt, endedAt time.Time, limit int, includeIdle bool) (semantic.ToolResult, error) {
	limit = normalizedLimit(limit, defaultTimelineEventLimit, maxTimelineEventLimit)
	events, err := r.filteredUsageEvents(ctx, startedAt, endedAt, includeIdle)
	if err != nil {
		return semantic.ToolResult{}, err
	}

	totalDuration := totalEventDuration(events, startedAt, endedAt)
	totalEventCount := len(events)
	truncated := totalEventCount > limit
	if truncated {
		events = events[:limit]
	}

	timelineEvents := make([]semantic.UsageTimelineEvent, 0, len(events))
	contentEvents := make([]timelineToolEvent, 0, len(events))
	for _, event := range events {
		timelineEvent := r.semanticTimelineEvent(event, startedAt, endedAt)
		timelineEvents = append(timelineEvents, timelineEvent)
		contentEvents = append(contentEvents, contentTimelineEvent(timelineEvent))
	}

	timeline := &semantic.UsageTimeline{
		StartedAt:            startedAt,
		EndedAt:              endedAt,
		TimeZone:             r.timeZone(),
		TotalDurationSeconds: durationSeconds(totalDuration),
		Events:               timelineEvents,
		TotalEventCount:      totalEventCount,
		Truncated:            truncated,
	}
	content, err := json.Marshal(usageToolContent{
		StartedAt:            startedAt.Format(time.RFC3339),
		EndedAt:              endedAt.Format(time.RFC3339),
		TimeZone:             timeline.TimeZone,
		TotalDurationSeconds: timeline.TotalDurationSeconds,
		Events:               contentEvents,
		Defaults: map[string]interface{}{
			"limit":        defaultTimelineEventLimit,
			"include_idle": false,
		},
	})
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("marshal tool result: %w", err)
	}

	return semantic.ToolResult{
		Content: string(content),
		Artifacts: []semantic.Artifact{{
			Type:          semantic.ArtifactTypeUsageTimeline,
			UsageTimeline: timeline,
		}},
	}, nil
}

func (r *AppUsageToolRunner) habitSummary(ctx context.Context, startedAt, endedAt time.Time, limit int, includeIdle bool) (semantic.ToolResult, error) {
	limit = normalizedLimit(limit, defaultAppUsageLimit, maxAppUsageLimit)
	events, err := r.filteredUsageEvents(ctx, startedAt, endedAt, includeIdle)
	if err != nil {
		return semantic.ToolResult{}, err
	}
	totals, err := r.usage.GetAppUsageTotals(ctx, componentUsage.AppUsageTotalsParams{
		Window: componentUsage.Window{
			StartedAt: startedAt,
			EndedAt:   endedAt,
		},
		Limit:       limit,
		IncludeIdle: includeIdle,
	})
	if err != nil {
		return semantic.ToolResult{}, err
	}

	totalDuration := totalEventDuration(events, startedAt, endedAt)
	var longest *semantic.UsageTimelineEvent
	var longestDuration time.Duration
	for _, event := range events {
		duration := clippedUsageDuration(event.StartedAt, event.EndedAt, startedAt, endedAt)
		if duration > longestDuration {
			timelineEvent := r.semanticTimelineEvent(event, startedAt, endedAt)
			longest = &timelineEvent
			longestDuration = duration
		}
	}

	summary := &semantic.UsageHabitSummary{
		StartedAt:             startedAt,
		EndedAt:               endedAt,
		TimeZone:              r.timeZone(),
		TotalDurationSeconds:  durationSeconds(totalDuration),
		SessionCount:          int64(len(events)),
		ContextSwitchCount:    contextSwitchCount(events),
		AverageSessionSeconds: averageDurationSeconds(totalDuration, len(events)),
		LongestSession:        longest,
		TopSources:            semanticBuckets(totals.Buckets),
		TimeBuckets:           r.timeOfDayBuckets(events, startedAt, endedAt),
	}

	var longestContent *timelineToolEvent
	if longest != nil {
		content := contentTimelineEvent(*longest)
		longestContent = &content
	}
	content, err := json.Marshal(usageToolContent{
		StartedAt:             startedAt.Format(time.RFC3339),
		EndedAt:               endedAt.Format(time.RFC3339),
		TimeZone:              summary.TimeZone,
		TotalDurationSeconds:  summary.TotalDurationSeconds,
		Buckets:               contentBucketsFromSemantic(summary.TopSources),
		SessionCount:          summary.SessionCount,
		ContextSwitchCount:    summary.ContextSwitchCount,
		AverageSessionSeconds: summary.AverageSessionSeconds,
		LongestSession:        longestContent,
		TimeBuckets:           contentTimeBuckets(summary.TimeBuckets),
		Defaults: map[string]interface{}{
			"limit":        defaultAppUsageLimit,
			"include_idle": false,
		},
	})
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("marshal tool result: %w", err)
	}

	return semantic.ToolResult{
		Content: string(content),
		Artifacts: []semantic.Artifact{{
			Type:              semantic.ArtifactTypeUsageHabitSummary,
			UsageHabitSummary: summary,
		}},
	}, nil
}

func (r *AppUsageToolRunner) usageComparison(
	ctx context.Context,
	currentStart time.Time,
	currentEnd time.Time,
	baselineStart time.Time,
	baselineEnd time.Time,
	limit int,
	includeIdle bool,
) (semantic.ToolResult, error) {
	limit = normalizedLimit(limit, defaultComparisonLimit, maxAppUsageLimit)
	current, err := r.usage.GetAppUsageTotals(ctx, componentUsage.AppUsageTotalsParams{
		Window:      componentUsage.Window{StartedAt: currentStart, EndedAt: currentEnd},
		IncludeIdle: includeIdle,
	})
	if err != nil {
		return semantic.ToolResult{}, err
	}
	baseline, err := r.usage.GetAppUsageTotals(ctx, componentUsage.AppUsageTotalsParams{
		Window:      componentUsage.Window{StartedAt: baselineStart, EndedAt: baselineEnd},
		IncludeIdle: includeIdle,
	})
	if err != nil {
		return semantic.ToolResult{}, err
	}

	buckets := comparisonBuckets(current.Buckets, baseline.Buckets, limit)
	comparison := &semantic.UsageComparison{
		CurrentStartedAt:             currentStart,
		CurrentEndedAt:               currentEnd,
		BaselineStartedAt:            baselineStart,
		BaselineEndedAt:              baselineEnd,
		TimeZone:                     r.timeZone(),
		CurrentTotalDurationSeconds:  durationSeconds(current.TotalDuration),
		BaselineTotalDurationSeconds: durationSeconds(baseline.TotalDuration),
		DurationDeltaSeconds:         durationSeconds(current.TotalDuration) - durationSeconds(baseline.TotalDuration),
		DurationDeltaPercent:         durationDeltaPercent(current.TotalDuration, baseline.TotalDuration),
		Buckets:                      buckets,
	}
	content, err := json.Marshal(usageToolContent{
		CurrentStartedAt:     currentStart.Format(time.RFC3339),
		CurrentEndedAt:       currentEnd.Format(time.RFC3339),
		BaselineStartedAt:    baselineStart.Format(time.RFC3339),
		BaselineEndedAt:      baselineEnd.Format(time.RFC3339),
		TimeZone:             comparison.TimeZone,
		CurrentTotalSeconds:  comparison.CurrentTotalDurationSeconds,
		BaselineTotalSeconds: comparison.BaselineTotalDurationSeconds,
		DurationDeltaSeconds: comparison.DurationDeltaSeconds,
		DurationDeltaPercent: comparison.DurationDeltaPercent,
		ComparisonBuckets:    contentComparisonBuckets(buckets),
		Defaults: map[string]interface{}{
			"limit":        defaultComparisonLimit,
			"include_idle": false,
		},
	})
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("marshal tool result: %w", err)
	}

	return semantic.ToolResult{
		Content: string(content),
		Artifacts: []semantic.Artifact{{
			Type:            semantic.ArtifactTypeUsageComparison,
			UsageComparison: comparison,
		}},
	}, nil
}

func (r *AppUsageToolRunner) filteredUsageEvents(ctx context.Context, startedAt, endedAt time.Time, includeIdle bool) ([]componentUsage.Event, error) {
	events, err := r.usage.GetEvents(ctx, componentUsage.GetEventsParams{
		Window: componentUsage.Window{
			StartedAt: startedAt,
			EndedAt:   endedAt,
		},
	})
	if err != nil {
		return nil, err
	}
	filtered := make([]componentUsage.Event, 0, len(events))
	for _, event := range events {
		if event.Source == componentUsage.SourceIdle && !includeIdle {
			continue
		}
		if clippedUsageDuration(event.StartedAt, event.EndedAt, startedAt, endedAt) <= 0 {
			continue
		}
		filtered = append(filtered, event)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].StartedAt.Equal(filtered[j].StartedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].StartedAt.Before(filtered[j].StartedAt)
	})
	return filtered, nil
}

func (r *AppUsageToolRunner) semanticTimelineEvent(event componentUsage.Event, windowStart, windowEnd time.Time) semantic.UsageTimelineEvent {
	startedAt := maxTime(event.StartedAt, windowStart)
	endedAt := minTime(event.EndedAt, windowEnd)
	return semantic.UsageTimelineEvent{
		TransitionEventID:     event.ID,
		Title:                 firstNonEmpty(event.Title, event.SourceName, event.ApplicationName, "Usage"),
		SourceName:            firstNonEmpty(event.SourceName, event.ApplicationName, "Application"),
		SourceType:            appUsageSourceType(event.Source),
		StartedAt:             startedAt,
		EndedAt:               endedAt,
		DurationSeconds:       durationSeconds(endedAt.Sub(startedAt)),
		ApplicationIdentifier: strings.TrimSpace(event.ApplicationIdentifier),
		ApplicationPath:       strings.TrimSpace(event.ApplicationPath),
		URLHost:               hostFromToolURL(event.CDPURL),
		Idle:                  event.Source == componentUsage.SourceIdle,
	}
}

func (r *AppUsageToolRunner) timeOfDayBuckets(events []componentUsage.Event, windowStart, windowEnd time.Time) []semantic.TimeOfDayBucket {
	buckets := []semantic.TimeOfDayBucket{
		{Label: "Overnight"},
		{Label: "Morning"},
		{Label: "Afternoon"},
		{Label: "Evening"},
	}
	for _, event := range events {
		start := maxTime(event.StartedAt, windowStart)
		end := minTime(event.EndedAt, windowEnd)
		visited := map[int]bool{}
		for start.Before(end) {
			index := timeBucketIndex(start.In(r.locationOrLocal()).Hour())
			next := minTime(end, nextTimeBucketBoundary(start, r.locationOrLocal()))
			buckets[index].DurationSeconds += durationSeconds(next.Sub(start))
			if !visited[index] {
				buckets[index].SessionCount++
				visited[index] = true
			}
			start = next
		}
	}
	return buckets
}

func semanticBuckets(buckets []componentUsage.AppUsageBucket) []semantic.AppUsageBucket {
	values := make([]semantic.AppUsageBucket, 0, len(buckets))
	for _, bucket := range buckets {
		values = append(values, semantic.AppUsageBucket{
			Name:                  bucket.Name,
			SourceType:            appUsageSourceType(bucket.Source),
			DurationSeconds:       durationSeconds(bucket.Duration),
			SessionCount:          int64(bucket.SessionCount),
			ApplicationIdentifier: bucket.ApplicationIdentifier,
			ApplicationPath:       bucket.ApplicationPath,
		})
	}
	return values
}

func contentBucketsFromSemantic(buckets []semantic.AppUsageBucket) []appUsageToolBucket {
	values := make([]appUsageToolBucket, 0, len(buckets))
	for _, bucket := range buckets {
		values = append(values, appUsageToolBucket{
			Name:                  bucket.Name,
			SourceType:            bucket.SourceType,
			DurationSeconds:       bucket.DurationSeconds,
			SessionCount:          bucket.SessionCount,
			ApplicationIdentifier: bucket.ApplicationIdentifier,
			ApplicationPath:       bucket.ApplicationPath,
		})
	}
	return values
}

func contentTimelineEvent(event semantic.UsageTimelineEvent) timelineToolEvent {
	return timelineToolEvent{
		TransitionEventID:     event.TransitionEventID,
		Title:                 event.Title,
		SourceName:            event.SourceName,
		SourceType:            event.SourceType,
		StartedAt:             event.StartedAt.Format(time.RFC3339),
		EndedAt:               event.EndedAt.Format(time.RFC3339),
		DurationSeconds:       event.DurationSeconds,
		ApplicationIdentifier: event.ApplicationIdentifier,
		ApplicationPath:       event.ApplicationPath,
		URLHost:               event.URLHost,
		Idle:                  event.Idle,
	}
}

func contentTimeBuckets(buckets []semantic.TimeOfDayBucket) []timeOfDayToolBucket {
	values := make([]timeOfDayToolBucket, 0, len(buckets))
	for _, bucket := range buckets {
		values = append(values, timeOfDayToolBucket{
			Label:           bucket.Label,
			DurationSeconds: bucket.DurationSeconds,
			SessionCount:    bucket.SessionCount,
		})
	}
	return values
}

func comparisonBuckets(currentBuckets, baselineBuckets []componentUsage.AppUsageBucket, limit int) []semantic.UsageComparisonBucket {
	type aggregate struct {
		name                    string
		sourceType              string
		currentDurationSeconds  int64
		baselineDurationSeconds int64
		currentSessionCount     int64
		baselineSessionCount    int64
	}
	byKey := map[string]*aggregate{}
	add := func(bucket componentUsage.AppUsageBucket, current bool) {
		sourceType := appUsageSourceType(bucket.Source)
		key := strings.ToLower(bucket.Name) + "\x00" + sourceType
		item := byKey[key]
		if item == nil {
			item = &aggregate{name: bucket.Name, sourceType: sourceType}
			byKey[key] = item
		}
		if current {
			item.currentDurationSeconds += durationSeconds(bucket.Duration)
			item.currentSessionCount += int64(bucket.SessionCount)
		} else {
			item.baselineDurationSeconds += durationSeconds(bucket.Duration)
			item.baselineSessionCount += int64(bucket.SessionCount)
		}
	}
	for _, bucket := range currentBuckets {
		add(bucket, true)
	}
	for _, bucket := range baselineBuckets {
		add(bucket, false)
	}

	values := make([]semantic.UsageComparisonBucket, 0, len(byKey))
	for _, item := range byKey {
		values = append(values, semantic.UsageComparisonBucket{
			Name:                    item.name,
			SourceType:              item.sourceType,
			CurrentDurationSeconds:  item.currentDurationSeconds,
			BaselineDurationSeconds: item.baselineDurationSeconds,
			DeltaDurationSeconds:    item.currentDurationSeconds - item.baselineDurationSeconds,
			CurrentSessionCount:     item.currentSessionCount,
			BaselineSessionCount:    item.baselineSessionCount,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		left := maxInt64(values[i].CurrentDurationSeconds, values[i].BaselineDurationSeconds)
		right := maxInt64(values[j].CurrentDurationSeconds, values[j].BaselineDurationSeconds)
		if left == right {
			return values[i].Name < values[j].Name
		}
		return left > right
	})
	if limit > 0 && len(values) > limit {
		values = values[:limit]
	}
	return values
}

func contentComparisonBuckets(buckets []semantic.UsageComparisonBucket) []comparisonToolBucket {
	values := make([]comparisonToolBucket, 0, len(buckets))
	for _, bucket := range buckets {
		values = append(values, comparisonToolBucket{
			Name:                    bucket.Name,
			SourceType:              bucket.SourceType,
			CurrentDurationSeconds:  bucket.CurrentDurationSeconds,
			BaselineDurationSeconds: bucket.BaselineDurationSeconds,
			DeltaDurationSeconds:    bucket.DeltaDurationSeconds,
			CurrentSessionCount:     bucket.CurrentSessionCount,
			BaselineSessionCount:    bucket.BaselineSessionCount,
		})
	}
	return values
}

func parseToolWindow(startedAtValue string, endedAtValue string) (time.Time, time.Time, error) {
	startedAt, err := parseToolTime(startedAtValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("started_at: %w", err)
	}
	endedAt, err := parseToolTime(endedAtValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("ended_at: %w", err)
	}
	if !endedAt.After(startedAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("ended_at must be after started_at")
	}
	return startedAt, endedAt, nil
}

func (r *AppUsageToolRunner) timeZone() string {
	if r != nil && r.location != nil {
		return r.location.String()
	}
	return time.Local.String()
}

func (r *AppUsageToolRunner) locationOrLocal() *time.Location {
	if r != nil && r.location != nil {
		return r.location
	}
	return time.Local
}

func parseToolTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func normalizedLimit(value int, fallback int, maximum int) int {
	if value <= 0 {
		value = fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func appUsageSourceType(source componentUsage.Source) string {
	switch source {
	case componentUsage.SourceBrowser:
		return "browser"
	case componentUsage.SourceIdle:
		return "idle"
	default:
		return "application"
	}
}

func totalEventDuration(events []componentUsage.Event, windowStart, windowEnd time.Time) time.Duration {
	var total time.Duration
	for _, event := range events {
		total += clippedUsageDuration(event.StartedAt, event.EndedAt, windowStart, windowEnd)
	}
	return total
}

func clippedUsageDuration(start, end, windowStart, windowEnd time.Time) time.Duration {
	start = maxTime(start, windowStart)
	end = minTime(end, windowEnd)
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}

func contextSwitchCount(events []componentUsage.Event) int64 {
	var count int64
	var previous string
	for _, event := range events {
		key := appUsageSourceType(event.Source) + "\x00" + firstNonEmpty(event.SourceName, event.ApplicationName, event.Title)
		if previous != "" && key != previous {
			count++
		}
		previous = key
	}
	return count
}

func averageDurationSeconds(total time.Duration, count int) int64 {
	if count <= 0 || total <= 0 {
		return 0
	}
	return durationSeconds(total / time.Duration(count))
}

func durationDeltaPercent(current time.Duration, baseline time.Duration) float64 {
	if baseline <= 0 {
		if current <= 0 {
			return 0
		}
		return 100
	}
	return roundPercent((float64(current-baseline) / float64(baseline)) * 100)
}

func durationSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	seconds := int64(duration / time.Second)
	if seconds == 0 {
		return 1
	}
	return seconds
}

func hostFromToolURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func timeBucketIndex(hour int) int {
	switch {
	case hour < 6:
		return 0
	case hour < 12:
		return 1
	case hour < 18:
		return 2
	default:
		return 3
	}
}

func nextTimeBucketBoundary(value time.Time, loc *time.Location) time.Time {
	local := value.In(loc)
	hour := local.Hour()
	nextHour := 6
	switch {
	case hour < 6:
		nextHour = 6
	case hour < 12:
		nextHour = 12
	case hour < 18:
		nextHour = 18
	default:
		return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	}
	return time.Date(local.Year(), local.Month(), local.Day(), nextHour, 0, 0, 0, loc)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func maxTime(first time.Time, second time.Time) time.Time {
	if first.After(second) {
		return first
	}
	return second
}

func minTime(first time.Time, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}

func maxInt64(first int64, second int64) int64 {
	if first > second {
		return first
	}
	return second
}

func roundPercent(value float64) float64 {
	return math.Round(value*10) / 10
}
