package semantic

import (
	"context"
	"fmt"
	"strings"
)

type DocumentSearcher interface {
	Search(ctx context.Context, query string, k int64) ([]SearchResult, error)
}

type Answerer struct {
	searcher  DocumentSearcher
	generator Generator
}

type Answer struct {
	Question string
	Answer   string
	Model    string
	Sources  []SearchResult
}

func NewAnswerer(searcher DocumentSearcher, generator Generator) *Answerer {
	return &Answerer{searcher: searcher, generator: generator}
}

func (a *Answerer) Answer(ctx context.Context, question string, k int64) (*Answer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive")
	}
	if a.searcher == nil {
		return nil, fmt.Errorf("searcher is required")
	}
	if a.generator == nil {
		return nil, fmt.Errorf("generator is required")
	}

	sources, err := a.searcher.Search(ctx, question, k)
	if err != nil {
		return nil, fmt.Errorf("search answer context: %w", err)
	}
	if len(sources) == 0 {
		return &Answer{
			Question: question,
			Answer:   "I could not find any relevant transition events to answer that question.",
			Model:    a.generator.Model(),
			Sources:  sources,
		}, nil
	}

	prompt := BuildAnswerPrompt(question, sources)
	answer, err := a.generator.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate answer: %w", err)
	}

	return &Answer{
		Question: question,
		Answer:   strings.TrimSpace(answer),
		Model:    a.generator.Model(),
		Sources:  sources,
	}, nil
}

func BuildAnswerPrompt(question string, sources []SearchResult) string {
	var b strings.Builder
	b.WriteString("You answer questions about indexed user activity history.\n")
	b.WriteString("Use only the context below. If the context is insufficient, say you do not have enough information.\n")
	b.WriteString("The context can contain event documents, day summaries, app-day summaries, and time-block summaries.\n")
	b.WriteString("Prefer summary documents for totals and broad questions. Use event documents for specific examples.\n")
	b.WriteString("Cite document_key values, and cite transition_event_id values when an event document supports your answer.\n")
	b.WriteString("Do not invent applications, tabs, URLs, dates, durations, or reasons that are not in the context.\n\n")
	b.WriteString("Question:\n")
	b.WriteString(strings.TrimSpace(question))
	b.WriteString("\n\nContext:\n")

	for i, source := range sources {
		b.WriteString(fmt.Sprintf("\n[%d] document_key=%s document_type=%s", i+1, source.DocumentKey, source.DocumentType))
		if source.TransitionEventID != 0 {
			b.WriteString(fmt.Sprintf(" transition_event_id=%d", source.TransitionEventID))
		}
		b.WriteString(fmt.Sprintf(" distance=%.6f\n", source.Distance))
		b.WriteString(strings.TrimSpace(source.Content))
		b.WriteString("\n")
	}

	b.WriteString("\nAnswer:")
	return b.String()
}
