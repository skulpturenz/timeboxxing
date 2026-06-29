package ama

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
)

const (
	appUsageToolName     = "get_app_usage_totals"
	defaultAppUsageLimit = 5
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

type appUsageToolContent struct {
	StartedAt            string                 `json:"started_at"`
	EndedAt              string                 `json:"ended_at"`
	TimeZone             string                 `json:"timezone"`
	TotalDurationSeconds int64                  `json:"total_duration_seconds"`
	Buckets              []appUsageToolBucket   `json:"buckets"`
	Defaults             map[string]interface{} `json:"defaults,omitempty"`
}

type appUsageToolBucket struct {
	Name                  string `json:"name"`
	SourceType            string `json:"source_type"`
	DurationSeconds       int64  `json:"duration_seconds"`
	SessionCount          int64  `json:"session_count"`
	ApplicationIdentifier string `json:"application_identifier,omitempty"`
	ApplicationPath       string `json:"application_path,omitempty"`
}

func NewAppUsageToolRunner(usage *componentUsage.Service) *AppUsageToolRunner {
	return &AppUsageToolRunner{
		usage:    usage,
		location: time.Local,
	}
}

func (r *AppUsageToolRunner) Tools() []semantic.ChatTool {
	return []semantic.ChatTool{{
		Name:        appUsageToolName,
		Description: "Returns total captured usage time per application for an exact time window. Browser activity is grouped under the browser application.",
		Parameters: map[string]any{
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
					"description": "Maximum number of app buckets to return. Default is 5.",
					"minimum":     1,
				},
				"include_idle": map[string]any{
					"type":        "boolean",
					"description": "Whether idle time should be included. Default false; only true when the user explicitly asks for idle time.",
				},
			},
			"required":             []string{"started_at", "ended_at"},
			"additionalProperties": false,
		},
	}}
}

func (r *AppUsageToolRunner) Execute(ctx context.Context, call semantic.ChatToolCall) (semantic.ToolResult, error) {
	if r == nil || r.usage == nil {
		return semantic.ToolResult{}, fmt.Errorf("usage service is required")
	}
	if strings.TrimSpace(call.Name) != appUsageToolName {
		return semantic.ToolResult{}, fmt.Errorf("unknown tool %q", call.Name)
	}

	var args appUsageToolArgs
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return semantic.ToolResult{}, fmt.Errorf("parse tool arguments: %w", err)
	}
	startedAt, err := parseToolTime(args.StartedAt)
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("started_at: %w", err)
	}
	endedAt, err := parseToolTime(args.EndedAt)
	if err != nil {
		return semantic.ToolResult{}, fmt.Errorf("ended_at: %w", err)
	}
	if !endedAt.After(startedAt) {
		return semantic.ToolResult{}, fmt.Errorf("ended_at must be after started_at")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = defaultAppUsageLimit
	}
	totals, err := r.usage.GetAppUsageTotals(ctx, componentUsage.AppUsageTotalsParams{
		Window: componentUsage.Window{
			StartedAt: startedAt,
			EndedAt:   endedAt,
		},
		Limit:       limit,
		IncludeIdle: args.IncludeIdle,
	})
	if err != nil {
		return semantic.ToolResult{}, err
	}

	zone := r.timeZone()
	buckets := make([]semantic.AppUsageBucket, 0, len(totals.Buckets))
	contentBuckets := make([]appUsageToolBucket, 0, len(totals.Buckets))
	for _, bucket := range totals.Buckets {
		semanticBucket := semantic.AppUsageBucket{
			Name:                  bucket.Name,
			SourceType:            appUsageSourceType(bucket.Source),
			DurationSeconds:       durationSeconds(bucket.Duration),
			SessionCount:          int64(bucket.SessionCount),
			ApplicationIdentifier: bucket.ApplicationIdentifier,
			ApplicationPath:       bucket.ApplicationPath,
		}
		buckets = append(buckets, semanticBucket)
		contentBuckets = append(contentBuckets, appUsageToolBucket{
			Name:                  semanticBucket.Name,
			SourceType:            semanticBucket.SourceType,
			DurationSeconds:       semanticBucket.DurationSeconds,
			SessionCount:          semanticBucket.SessionCount,
			ApplicationIdentifier: semanticBucket.ApplicationIdentifier,
			ApplicationPath:       semanticBucket.ApplicationPath,
		})
	}

	chart := &semantic.AppUsageChart{
		StartedAt:            startedAt,
		EndedAt:              endedAt,
		TimeZone:             zone,
		TotalDurationSeconds: durationSeconds(totals.TotalDuration),
		Buckets:              buckets,
	}
	content, err := json.Marshal(appUsageToolContent{
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
