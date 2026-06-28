package semantic

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestAnswererSearchesAndGeneratesGroundedAnswer(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeDocumentSearcher{
		results: []SearchResult{
			{
				TransitionEventID: 42,
				DocumentKey:       "event:42",
				DocumentType:      DocumentTypeEvent,
				Content:           "Reason: tab_change\nApplication: Google Chrome\nTab: Pull request review\nURL: https://github.com/skulpturenz/timeboxxing/pull/1",
				Distance:          0.125,
			},
			{
				TransitionEventID: 43,
				DocumentKey:       "event:43",
				DocumentType:      DocumentTypeEvent,
				Content:           "Reason: focus_change\nApplication: VSCode",
				Distance:          0.25,
			},
		},
	}
	generator := &fakeGenerator{response: "You reviewed a GitHub pull request, then switched to VSCode. [transition_event_id=42, transition_event_id=43]"}
	answerer := NewAnswerer(searcher, generator)

	answer, err := answerer.Answer(ctx, " What was I doing? ", 2)
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}

	if searcher.query != "What was I doing?" || searcher.k != 2 {
		t.Fatalf("unexpected search call: query=%q k=%d", searcher.query, searcher.k)
	}
	if answer.Question != "What was I doing?" {
		t.Fatalf("unexpected question %q", answer.Question)
	}
	if answer.Model != "google/gemma-4-31b-it:free" {
		t.Fatalf("unexpected model %q", answer.Model)
	}
	if answer.Answer != generator.response {
		t.Fatalf("unexpected answer %q", answer.Answer)
	}
	if len(answer.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(answer.Sources))
	}

	assertContains(t, generator.prompt, "Use only the context below")
	assertContains(t, generator.prompt, "If the context is insufficient")
	assertContains(t, generator.prompt, "Prefer summary documents for totals")
	assertContains(t, generator.prompt, "Do not invent applications")
	assertContains(t, generator.prompt, "Question:\nWhat was I doing?")
	assertContains(t, generator.prompt, "document_key=event:42")
	assertContains(t, generator.prompt, "transition_event_id=42")
	assertContains(t, generator.prompt, "distance=0.125000")
	assertContains(t, generator.prompt, "Application: Google Chrome")
	assertContains(t, generator.prompt, "document_key=event:43")
	assertContains(t, generator.prompt, "transition_event_id=43")
	assertContains(t, generator.prompt, "Application: VSCode")
}

func TestAnswererReturnsNoContextAnswerWithoutGenerating(t *testing.T) {
	answerer := NewAnswerer(&fakeDocumentSearcher{}, &fakeGenerator{})

	answer, err := answerer.Answer(context.Background(), "What did I do yesterday?", 3)
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}
	if answer.Answer != "I could not find any relevant transition events to answer that question." {
		t.Fatalf("unexpected answer %q", answer.Answer)
	}
	if len(answer.Sources) != 0 {
		t.Fatalf("expected no sources, got %d", len(answer.Sources))
	}
}

func TestAnswererValidation(t *testing.T) {
	answerer := NewAnswerer(&fakeDocumentSearcher{}, &fakeGenerator{})

	tests := []struct {
		name     string
		question string
		k        int64
		want     string
	}{
		{name: "empty question", question: " ", k: 1, want: "question is required"},
		{name: "invalid k", question: "hello", k: 0, want: "k must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := answerer.Answer(context.Background(), tt.question, tt.k)
			if err == nil {
				t.Fatal("expected error")
			}
			assertContains(t, err.Error(), tt.want)
		})
	}
}

func TestAnswererPropagatesSearchAndGenerationErrors(t *testing.T) {
	_, err := NewAnswerer(&fakeDocumentSearcher{err: fmt.Errorf("search failed")}, &fakeGenerator{}).Answer(context.Background(), "question", 1)
	if err == nil {
		t.Fatal("expected search error")
	}
	assertContains(t, err.Error(), "search answer context")
	assertContains(t, err.Error(), "search failed")

	_, err = NewAnswerer(&fakeDocumentSearcher{results: []SearchResult{{TransitionEventID: 1, Content: "Application: VSCode"}}}, &fakeGenerator{err: fmt.Errorf("gemma failed")}).Answer(context.Background(), "question", 1)
	if err == nil {
		t.Fatal("expected generation error")
	}
	assertContains(t, err.Error(), "generate answer")
	assertContains(t, err.Error(), "gemma failed")
}

func TestBuildAnswerPrompt(t *testing.T) {
	prompt := BuildAnswerPrompt("Where was I browsing?", []SearchResult{
		{TransitionEventID: 7, DocumentKey: "event:7", DocumentType: DocumentTypeEvent, Content: "Application: Google Chrome\nTab: Docs", Distance: 0.5},
	})

	assertContains(t, prompt, "You answer questions about indexed user activity history.")
	assertContains(t, prompt, "The context can contain event documents")
	assertContains(t, prompt, "Question:\nWhere was I browsing?")
	assertContains(t, prompt, "[1] document_key=event:7 document_type=event transition_event_id=7 distance=0.500000")
	assertContains(t, prompt, "Application: Google Chrome")
	assertContains(t, prompt, "Answer:")
}

type fakeDocumentSearcher struct {
	query   string
	k       int64
	results []SearchResult
	err     error
}

func (f *fakeDocumentSearcher) Search(_ context.Context, query string, k int64) ([]SearchResult, error) {
	f.query = query
	f.k = k
	return f.results, f.err
}

type fakeGenerator struct {
	prompt   string
	response string
	err      error
}

func (f *fakeGenerator) Model() string { return "google/gemma-4-31b-it:free" }

func (f *fakeGenerator) Generate(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.response, f.err
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}
