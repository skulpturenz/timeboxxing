package semantic

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A usage-shaped question is routed to the timeline tool by the model, which infers the window from
// the question: there is no hand-rolled period parser any more.
func TestAnswererRoutesUsageQuestionsToTheTimelineTool(t *testing.T) {
	startedAt := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.AddDate(0, 0, 1)
	searcher := &fakeDocumentSearcher{
		results: []SearchResult{{Content: "Application: Should not be used"}},
	}
	generator := &fakeToolCallingGenerator{
		responses: []ChatResponse{
			{ToolCalls: []ChatToolCall{{
				ID:        "call-1",
				Name:      "get_usage_timeline",
				Arguments: `{"started_at":"2026-06-29T00:00:00Z","ended_at":"2026-06-30T00:00:00Z"}`,
			}}},
			{Content: "You spent yesterday in Chrome."},
		},
	}
	toolRunner := &fakeToolRunner{
		result: ToolResult{
			Content: `{"events":[{"title":"Docs"}]}`,
			Artifacts: []Artifact{{
				Type: ArtifactTypeUsageTimeline,
				UsageTimeline: &UsageTimeline{
					StartedAt:            startedAt,
					EndedAt:              endedAt,
					TimeZone:             "UTC",
					TotalDurationSeconds: 3600,
					TotalEventCount:      1,
				},
			}},
		},
	}
	answerer := NewAnswerer(searcher, generator)
	answerer.SetToolRunner(toolRunner)

	answer, err := answerer.Answer(context.Background(), "Which applications did I use the most yesterday?", 5)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}

	if searcher.query != "" {
		t.Fatalf("expected the tool route to skip semantic search, got query %q", searcher.query)
	}
	if toolRunner.call.Name != "get_usage_timeline" {
		t.Fatalf("expected the timeline tool to be called, got %#v", toolRunner.call)
	}
	if answer.Answer != "You spent yesterday in Chrome." {
		t.Fatalf("unexpected answer %q", answer.Answer)
	}
	if len(answer.Artifacts) != 1 || answer.Artifacts[0].UsageTimeline == nil {
		t.Fatalf("expected a timeline artifact, got %#v", answer.Artifacts)
	}
}

// With no answer text from the model, the artifact itself has to say something useful.
func TestAnswererFallsBackToTheArtifactWhenTheModelSaysNothing(t *testing.T) {
	searcher := &fakeDocumentSearcher{}
	generator := &fakeToolCallingGenerator{
		responses: []ChatResponse{
			{ToolCalls: []ChatToolCall{{ID: "call-1", Name: "get_usage_timeline", Arguments: `{}`}}},
			{Content: "   "},
		},
	}
	toolRunner := &fakeToolRunner{
		result: ToolResult{
			Artifacts: []Artifact{{
				Type:          ArtifactTypeUsageTimeline,
				UsageTimeline: &UsageTimeline{},
			}},
		},
	}
	answerer := NewAnswerer(searcher, generator)
	answerer.SetToolRunner(toolRunner)

	answer, err := answerer.Answer(context.Background(), "Which apps did I use the most?", 5)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if answer.Answer != "I did not find any usage events in that period." {
		t.Fatalf("unexpected fallback answer %q", answer.Answer)
	}
}

func TestAnswererFallsBackToRAGForNonAppUsageQuestion(t *testing.T) {
	searcher := &fakeDocumentSearcher{
		results: []SearchResult{{TransitionEventID: 1, Content: "Application: Calendar"}},
	}
	generator := &fakeToolCallingGenerator{
		responses: []ChatResponse{{Content: "unused tool response"}},
		generate:  "Calendar work.",
	}
	answerer := NewAnswerer(searcher, generator)
	answerer.SetToolRunner(&fakeToolRunner{})

	answer, err := answerer.Answer(context.Background(), "What did I do today?", 5)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}

	if len(generator.messages) != 0 {
		t.Fatalf("expected no tool decision for non-app question")
	}
	if searcher.query != "What did I do today?" {
		t.Fatalf("expected RAG search, got %q", searcher.query)
	}
	if answer.Answer != "Calendar work." {
		t.Fatalf("unexpected answer %q", answer.Answer)
	}
}

func TestAnswererStructuredQueryBypassesSearchAndToolDecision(t *testing.T) {
	startedAt := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)
	searcher := &fakeDocumentSearcher{
		results: []SearchResult{{TransitionEventID: 99, Content: "Application: Should not be used"}},
	}
	generator := &fakeToolCallingGenerator{}
	toolRunner := &fakeStructuredToolRunner{
		result: ToolResult{
			Artifacts: []Artifact{{
				Type: ArtifactTypeUsageTimeline,
				UsageTimeline: &UsageTimeline{
					StartedAt:            startedAt,
					EndedAt:              endedAt,
					TimeZone:             "UTC",
					TotalDurationSeconds: 600,
					TotalEventCount:      1,
					Events: []UsageTimelineEvent{{
						TransitionEventID: 42,
						Title:             "Docs",
						SourceName:        "Google Chrome",
						SourceType:        "browser",
						StartedAt:         startedAt,
						EndedAt:           startedAt.Add(10 * time.Minute),
						DurationSeconds:   600,
						URLHost:           "example.com",
					}},
				},
			}},
		},
	}
	answerer := NewAnswerer(searcher, generator)
	answerer.SetToolRunner(toolRunner)

	answer, err := answerer.AnswerStructured(context.Background(), StructuredQuery{
		Kind:        StructuredQueryKindTimeline,
		Window:      TimeWindow{StartedAt: startedAt, EndedAt: endedAt},
		Limit:       10,
		PeriodLabel: "Today",
	})
	if err != nil {
		t.Fatalf("answer structured: %v", err)
	}

	if searcher.query != "" {
		t.Fatalf("expected structured route to skip semantic search, got query %q", searcher.query)
	}
	if len(generator.messages) != 0 {
		t.Fatalf("expected structured route to skip model tool decision, got %d calls", len(generator.messages))
	}
	if toolRunner.query.Kind != StructuredQueryKindTimeline || toolRunner.query.PeriodLabel != "Today" {
		t.Fatalf("unexpected structured query passed to runner: %#v", toolRunner.query)
	}
	if answer.Question != "Timeline for Today" {
		t.Fatalf("unexpected question label %q", answer.Question)
	}
	if !strings.Contains(answer.Answer, "I found 1 usage events for Today") {
		t.Fatalf("unexpected answer %q", answer.Answer)
	}
	if got := answer.Artifacts[0].UsageTimeline.PeriodLabel; got != "Today" {
		t.Fatalf("expected structured period label to be applied, got %q", got)
	}
}

type fakeToolCallingGenerator struct {
	responses []ChatResponse
	messages  [][]ChatMessage
	tools     [][]ChatTool
	generate  string
}

func (f *fakeToolCallingGenerator) Model() string { return "tool-generator" }

func (f *fakeToolCallingGenerator) Generate(_ context.Context, _ string) (string, error) {
	if f.generate != "" {
		return f.generate, nil
	}
	return "generated", nil
}

func (f *fakeToolCallingGenerator) GenerateWithTools(_ context.Context, messages []ChatMessage, tools []ChatTool) (ChatResponse, error) {
	f.messages = append(f.messages, messages)
	f.tools = append(f.tools, tools)
	if len(f.responses) == 0 {
		return ChatResponse{}, nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

type fakeToolRunner struct {
	call   ChatToolCall
	result ToolResult
}

func (f *fakeToolRunner) Tools() []ChatTool {
	return []ChatTool{{Name: "get_usage_timeline"}}
}

func (f *fakeToolRunner) Execute(_ context.Context, call ChatToolCall) (ToolResult, error) {
	f.call = call
	return f.result, nil
}

type fakeStructuredToolRunner struct {
	fakeToolRunner
	query  StructuredQuery
	result ToolResult
}

func (f *fakeStructuredToolRunner) ExecuteStructured(_ context.Context, query StructuredQuery) (ToolResult, error) {
	f.query = query
	return f.result, nil
}
