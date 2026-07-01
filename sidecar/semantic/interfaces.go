package semantic

import (
	"context"
	"time"
)

type Embedder interface {
	Model() string
	Dimension() int
	Embed(ctx context.Context, input string) ([]float32, error)
}

type Generator interface {
	Model() string
	Generate(ctx context.Context, prompt string) (string, error)
}

type ToolCallingGenerator interface {
	Model() string
	GenerateWithTools(ctx context.Context, messages []ChatMessage, tools []ChatTool) (ChatResponse, error)
}

type ChatRole string

const (
	ChatRoleSystem    ChatRole = "system"
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleTool      ChatRole = "tool"
)

type ChatMessage struct {
	Role       ChatRole
	Content    string
	ToolCallID string
	ToolCalls  []ChatToolCall
}

type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ChatToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ChatResponse struct {
	Content   string
	ToolCalls []ChatToolCall
}

type ToolRunner interface {
	Tools() []ChatTool
	Execute(ctx context.Context, call ChatToolCall) (ToolResult, error)
}

type StructuredToolRunner interface {
	ToolRunner
	ExecuteStructured(ctx context.Context, query StructuredQuery) (ToolResult, error)
}

type StructuredQueryKind string

const (
	StructuredQueryKindAppTotals      StructuredQueryKind = "app_totals"
	StructuredQueryKindTimeline       StructuredQueryKind = "timeline"
	StructuredQueryKindHabits         StructuredQueryKind = "habits"
	StructuredQueryKindComparePeriods StructuredQueryKind = "compare_periods"
)

type TimeWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

type StructuredQuery struct {
	Kind                StructuredQueryKind
	Window              TimeWindow
	BaselineWindow      TimeWindow
	Limit               int
	IncludeIdle         bool
	PeriodLabel         string
	BaselinePeriodLabel string
}

type ToolResult struct {
	Content   string
	Artifacts []Artifact
}

type ArtifactType string

const (
	ArtifactTypeAppUsageChart     ArtifactType = "app_usage_chart"
	ArtifactTypeUsageTimeline     ArtifactType = "usage_timeline"
	ArtifactTypeUsageHabitSummary ArtifactType = "usage_habit_summary"
	ArtifactTypeUsageComparison   ArtifactType = "usage_comparison"
)

type Artifact struct {
	Type              ArtifactType
	AppUsageChart     *AppUsageChart
	UsageTimeline     *UsageTimeline
	UsageHabitSummary *UsageHabitSummary
	UsageComparison   *UsageComparison
}

type AppUsageChart struct {
	PeriodLabel          string
	StartedAt            time.Time
	EndedAt              time.Time
	TimeZone             string
	TotalDurationSeconds int64
	Buckets              []AppUsageBucket
}

type AppUsageBucket struct {
	Name                  string
	SourceType            string
	DurationSeconds       int64
	SessionCount          int64
	ApplicationIdentifier string
	ApplicationPath       string
}

type UsageTimeline struct {
	PeriodLabel          string
	StartedAt            time.Time
	EndedAt              time.Time
	TimeZone             string
	TotalDurationSeconds int64
	Events               []UsageTimelineEvent
	TotalEventCount      int
	Truncated            bool
}

type UsageTimelineEvent struct {
	TransitionEventID     int64
	Title                 string
	SourceName            string
	SourceType            string
	StartedAt             time.Time
	EndedAt               time.Time
	DurationSeconds       int64
	ApplicationIdentifier string
	ApplicationPath       string
	URLHost               string
	Idle                  bool
}

type UsageHabitSummary struct {
	PeriodLabel           string
	StartedAt             time.Time
	EndedAt               time.Time
	TimeZone              string
	TotalDurationSeconds  int64
	SessionCount          int64
	ContextSwitchCount    int64
	AverageSessionSeconds int64
	LongestSession        *UsageTimelineEvent
	TopSources            []AppUsageBucket
	TimeBuckets           []TimeOfDayBucket
}

type TimeOfDayBucket struct {
	Label           string
	DurationSeconds int64
	SessionCount    int64
}

type UsageComparison struct {
	CurrentStartedAt             time.Time
	CurrentEndedAt               time.Time
	BaselineStartedAt            time.Time
	BaselineEndedAt              time.Time
	TimeZone                     string
	CurrentPeriodLabel           string
	BaselinePeriodLabel          string
	CurrentTotalDurationSeconds  int64
	BaselineTotalDurationSeconds int64
	DurationDeltaSeconds         int64
	DurationDeltaPercent         float64
	Buckets                      []UsageComparisonBucket
}

type UsageComparisonBucket struct {
	Name                    string
	SourceType              string
	CurrentDurationSeconds  int64
	BaselineDurationSeconds int64
	DeltaDurationSeconds    int64
	CurrentSessionCount     int64
	BaselineSessionCount    int64
}
