package ama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelinemodels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

const (
	usageTimelineToolName = "get_usage_timeline"

	defaultTimelineEventLimit = 50
	maxTimelineEventLimit     = 100

	// timelinePageSize is how many entries each keyset page of a timeline read carries.
	timelinePageSize = 256
)

type AppUsageToolRunner struct {
	registry *services.Services[any, any]
	location *time.Location
}

type timelineToolArgs struct {
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at"`
	Limit       int    `json:"limit"`
	IncludeIdle bool   `json:"include_idle"`
}

type usageToolContent struct {
	StartedAt            string                 `json:"started_at,omitempty"`
	EndedAt              string                 `json:"ended_at,omitempty"`
	TimeZone             string                 `json:"timezone,omitempty"`
	TotalDurationSeconds int64                  `json:"total_duration_seconds,omitempty"`
	Events               []timelineToolEvent    `json:"events,omitempty"`
	Defaults             map[string]interface{} `json:"defaults,omitempty"`
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

func NewAppUsageToolRunner(registry *services.Services[any, any]) *AppUsageToolRunner {
	return &AppUsageToolRunner{
		registry: registry,
		location: time.Local,
	}
}

func (r *AppUsageToolRunner) Tools() []semantic.ChatTool {
	return []semantic.ChatTool{
		{
			Name:        usageTimelineToolName,
			Description: "Returns exact usage transition events for an exact time window, including transition ids, titles, sources, durations, and URL hosts only.",
			Parameters:  usageWindowToolParameters(defaultTimelineEventLimit),
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

func (r *AppUsageToolRunner) Execute(ctx context.Context, call semantic.ChatToolCall) (semantic.ToolResult, error) {
	if r == nil || r.registry == nil {
		return semantic.ToolResult{}, fmt.Errorf("usage service is required")
	}

	switch strings.TrimSpace(call.Name) {
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

	default:
		return semantic.ToolResult{}, fmt.Errorf("unknown tool %q", call.Name)
	}
}

func (r *AppUsageToolRunner) ExecuteStructured(ctx context.Context, query semantic.StructuredQuery) (semantic.ToolResult, error) {
	if r == nil || r.registry == nil {
		return semantic.ToolResult{}, fmt.Errorf("usage service is required")
	}
	if !query.Window.EndedAt.After(query.Window.StartedAt) {
		return semantic.ToolResult{}, fmt.Errorf("ended_at must be after started_at")
	}

	switch query.Kind {
	case semantic.StructuredQueryKindTimeline:
		return r.usageTimeline(ctx, query.Window.StartedAt, query.Window.EndedAt, query.Limit, query.IncludeIdle)
	default:
		return semantic.ToolResult{}, fmt.Errorf("unsupported structured query kind %q", query.Kind)
	}
}

func (r *AppUsageToolRunner) usageTimeline(ctx context.Context, startedAt, endedAt time.Time, limit int, includeIdle bool) (semantic.ToolResult, error) {
	limit = normalizedLimit(limit, defaultTimelineEventLimit, maxTimelineEventLimit)
	window := utils.TimeSpan{startedAt, endedAt}
	entries, err := r.filteredUsageEntries(ctx, window, includeIdle)
	if err != nil {
		return semantic.ToolResult{}, err
	}

	totalDuration := totalEntryDuration(entries, window)
	totalEventCount := len(entries)
	truncated := totalEventCount > limit
	if truncated {
		entries = entries[:limit]
	}

	timelineEvents := make([]semantic.UsageTimelineEvent, 0, len(entries))
	contentEvents := make([]timelineToolEvent, 0, len(entries))
	for _, entry := range entries {
		timelineEvent := semanticTimelineEvent(entry, window)
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

// filteredUsageEntries reports the window's recorded stretches, oldest first. An entry still open has
// not been recorded as a stretch yet, and one that contributes nothing to the window is not worth
// reporting.
func (r *AppUsageToolRunner) filteredUsageEntries(ctx context.Context, window utils.TimeSpan, includeIdle bool) ([]timelinemodels.UsageSeq, error) {
	query := &componentTimeline.QueryGetTimelineRange{
		StartedAt: window[0],
		EndedAt:   window[1],
	}
	// the stream ends of its own accord once the window is drained, so the only way this comes back
	// short is a cancelled ctx — which is an error, not an empty window
	entries := slices.Collect(utils.SeqChan(utils.Stream(ctx, timelinePageSize, query.Stream(ctx, r.registry))))
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	filtered := make([]timelinemodels.UsageSeq, 0, len(entries))
	for _, entry := range entries {
		if entry.Start == nil || entry.End == nil {
			continue
		}
		if entry.Start.Idle && !includeIdle {
			continue
		}
		if entry.Span().ClippedDuration(window) <= 0 {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered, nil
}

func semanticTimelineEvent(entry timelinemodels.UsageSeq, window utils.TimeSpan) semantic.UsageTimelineEvent {
	span, _ := entry.Span().Clip(window)
	start := entry.Start

	return semantic.UsageTimelineEvent{
		TransitionEventID:     entry.ID,
		Title:                 firstNonEmpty(start.Title(), start.SourceName(), start.ApplicationName(), "Usage"),
		SourceName:            firstNonEmpty(start.SourceName(), start.ApplicationName(), "Application"),
		SourceType:            usageSourceType(*start),
		StartedAt:             span[0],
		EndedAt:               span[1],
		DurationSeconds:       durationSeconds(span.Duration()),
		ApplicationIdentifier: strings.TrimSpace(utils.Coalesce(start.AppIdentifier, "")),
		ApplicationPath:       strings.TrimSpace(utils.Coalesce(start.AppPath, "")),
		URLHost:               hostFromToolURL(start.Enrichments.Browser.CdpURL),
		Idle:                  start.Idle,
	}
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

func usageSourceType(foregroundProcess timelinemodels.ForegroundProcess) string {
	switch {
	case foregroundProcess.Idle:
		return "idle"
	case foregroundProcess.IsBrowser():
		return "browser"
	default:
		return "application"
	}
}

func totalEntryDuration(entries []timelinemodels.UsageSeq, window utils.TimeSpan) time.Duration {
	var total time.Duration
	for _, entry := range entries {
		total += entry.Span().ClippedDuration(window)
	}
	return total
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
