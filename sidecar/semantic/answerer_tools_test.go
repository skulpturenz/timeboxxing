package semantic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAnswererRoutesAppUsageQuestionsDirectlyToChartTool(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 6, 30, 10, 0, 0, 0, loc)
	tests := []struct {
		name        string
		question    string
		wantStart   time.Time
		wantEnd     time.Time
		wantLabel   string
		wantIdle    bool
		wantSnippet string
	}{
		{
			name:        "applications use the most yesterday",
			question:    "Which applications did I use the most yesterday?",
			wantStart:   time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantEnd:     time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantLabel:   "Yesterday",
			wantSnippet: "Your most used app for Yesterday was Chrome",
		},
		{
			name:        "applications use most yesterday",
			question:    "Which applications did I use most yesterday?",
			wantStart:   time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantEnd:     time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantLabel:   "Yesterday",
			wantSnippet: "Your most used app for Yesterday was Chrome",
		},
		{
			name:        "top apps today",
			question:    "What were my top apps today?",
			wantStart:   time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantEnd:     time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantLabel:   "Today",
			wantSnippet: "Your most used app for Today was Chrome",
		},
		{
			name:        "time by app last week",
			question:    "How much time did I spend by app last week?",
			wantStart:   time.Date(2026, 6, 22, 0, 0, 0, 0, loc),
			wantEnd:     time.Date(2026, 6, 29, 0, 0, 0, 0, loc),
			wantLabel:   "Last week",
			wantSnippet: "Your most used app for Last week was Chrome",
		},
		{
			name:        "idle opt in",
			question:    "What were my top apps including idle today?",
			wantStart:   time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
			wantEnd:     time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantLabel:   "Today",
			wantIdle:    true,
			wantSnippet: "Your most used app for Today was Chrome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &fakeDocumentSearcher{
				results: []SearchResult{{Content: "Application: Should not be used"}},
			}
			generator := &fakeToolCallingGenerator{}
			toolRunner := &fakeToolRunner{
				result: ToolResult{
					Content: `{"buckets":[{"name":"Chrome","duration_seconds":3600}]}`,
					Artifacts: []Artifact{{
						Type: ArtifactTypeAppUsageChart,
						AppUsageChart: &AppUsageChart{
							StartedAt:            tt.wantStart,
							EndedAt:              tt.wantEnd,
							TimeZone:             "UTC",
							TotalDurationSeconds: 3600,
							Buckets: []AppUsageBucket{{
								Name:            "Chrome",
								SourceType:      "browser",
								DurationSeconds: 3600,
								SessionCount:    1,
							}},
						},
					}},
				},
			}
			answerer := NewAnswerer(searcher, generator)
			answerer.clock = func() time.Time { return now }
			answerer.location = loc
			answerer.SetToolRunner(toolRunner)

			answer, err := answerer.Answer(context.Background(), tt.question, 5)
			if err != nil {
				t.Fatalf("answer: %v", err)
			}

			if searcher.query != "" {
				t.Fatalf("expected chart route to skip semantic search, got query %q", searcher.query)
			}
			if len(generator.messages) != 0 {
				t.Fatalf("expected chart route to skip model tool decision, got %d calls", len(generator.messages))
			}
			if toolRunner.call.Name != "get_app_usage_totals" {
				t.Fatalf("expected usage tool call, got %#v", toolRunner.call)
			}
			var args appUsageToolArgs
			if err := json.Unmarshal([]byte(toolRunner.call.Arguments), &args); err != nil {
				t.Fatalf("parse tool args: %v", err)
			}
			if args.StartedAt != tt.wantStart.Format(time.RFC3339) || args.EndedAt != tt.wantEnd.Format(time.RFC3339) {
				t.Fatalf("unexpected time window: %+v", args)
			}
			if args.Limit != 5 {
				t.Fatalf("expected default top 5 limit, got %d", args.Limit)
			}
			if args.IncludeIdle != tt.wantIdle {
				t.Fatalf("unexpected include_idle: %+v", args)
			}
			if len(answer.Artifacts) != 1 || answer.Artifacts[0].AppUsageChart == nil {
				t.Fatalf("expected chart artifact, got %#v", answer.Artifacts)
			}
			if answer.Artifacts[0].AppUsageChart.PeriodLabel != tt.wantLabel {
				t.Fatalf("expected period label %q, got %q", tt.wantLabel, answer.Artifacts[0].AppUsageChart.PeriodLabel)
			}
			if !strings.Contains(answer.Answer, tt.wantSnippet) {
				t.Fatalf("expected %q to contain %q", answer.Answer, tt.wantSnippet)
			}
		})
	}
}

func TestAnswererClarifiesAppUsageQuestionWithoutPeriod(t *testing.T) {
	searcher := &fakeDocumentSearcher{
		results: []SearchResult{{Content: "Application: Should not be used"}},
	}
	generator := &fakeToolCallingGenerator{}
	toolRunner := &fakeToolRunner{}
	answerer := NewAnswerer(searcher, generator)
	answerer.SetToolRunner(toolRunner)

	answer, err := answerer.Answer(context.Background(), "Which apps did I use the most?", 5)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if searcher.query != "" {
		t.Fatalf("expected no semantic search, got query %q", searcher.query)
	}
	if toolRunner.call.Name != "" {
		t.Fatalf("expected no tool call without a period, got %#v", toolRunner.call)
	}
	if answer.Answer != "Which time period should I chart for app usage?" {
		t.Fatalf("unexpected clarification %q", answer.Answer)
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
	return []ChatTool{{Name: "get_app_usage_totals"}}
}

func (f *fakeToolRunner) Execute(_ context.Context, call ChatToolCall) (ToolResult, error) {
	f.call = call
	return f.result, nil
}
