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
	StructuredQueryKindTimeline StructuredQueryKind = "timeline"
)

type TimeWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

type StructuredQuery struct {
	Kind        StructuredQueryKind
	Window      TimeWindow
	Limit       int
	IncludeIdle bool
	PeriodLabel string
}

type ToolResult struct {
	Content   string
	Artifacts []Artifact
}

type ArtifactType string

const (
	ArtifactTypeUsageTimeline ArtifactType = "usage_timeline"
)

type Artifact struct {
	Type          ArtifactType
	UsageTimeline *UsageTimeline
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
