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

type ToolResult struct {
	Content   string
	Artifacts []Artifact
}

type ArtifactType string

const (
	ArtifactTypeAppUsageChart ArtifactType = "app_usage_chart"
)

type Artifact struct {
	Type          ArtifactType
	AppUsageChart *AppUsageChart
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
