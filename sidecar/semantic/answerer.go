package semantic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var applicationWordRegexp = regexp.MustCompile(`\b(app|apps|application|applications|program|programs|software)\b`)

type DocumentSearcher interface {
	Search(ctx context.Context, query string, k int64) ([]SearchResult, error)
}

type Answerer struct {
	searcher   DocumentSearcher
	generator  Generator
	toolRunner ToolRunner
	clock      func() time.Time
	location   *time.Location
}

type Answer struct {
	Question  string
	Answer    string
	Model     string
	Sources   []SearchResult
	Artifacts []Artifact
}

func NewAnswerer(searcher DocumentSearcher, generator Generator) *Answerer {
	return &Answerer{
		searcher:  searcher,
		generator: generator,
		clock:     time.Now,
		location:  time.Local,
	}
}

func (a *Answerer) SetToolRunner(toolRunner ToolRunner) {
	a.toolRunner = toolRunner
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

	if toolAnswer, handled, err := a.answerWithTools(ctx, question); handled || err != nil {
		return toolAnswer, err
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

func (a *Answerer) AnswerStructured(ctx context.Context, query StructuredQuery) (*Answer, error) {
	if query.Kind == "" {
		return nil, fmt.Errorf("structured query kind is required")
	}
	if !query.Window.EndedAt.After(query.Window.StartedAt) {
		return nil, fmt.Errorf("structured query window must end after it starts")
	}
	if a.generator == nil {
		return nil, fmt.Errorf("generator is required")
	}
	runner, ok := a.toolRunner.(StructuredToolRunner)
	if !ok || runner == nil {
		return &Answer{
			Question: structuredQuestionLabel(query),
			Answer:   "I can answer structured usage questions once the usage insights tools are available.",
			Model:    a.generator.Model(),
		}, nil
	}

	result, err := runner.ExecuteStructured(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("execute structured usage query: %w", err)
	}
	applyStructuredLabels(result.Artifacts, query)

	return &Answer{
		Question:  structuredQuestionLabel(query),
		Answer:    structuredUsageAnswer(result.Artifacts, query),
		Model:     a.generator.Model(),
		Artifacts: result.Artifacts,
	}, nil
}

func (a *Answerer) answerWithTools(ctx context.Context, question string) (*Answer, bool, error) {
	if a.toolRunner == nil || !looksLikeAppUsageQuestion(question) {
		return nil, false, nil
	}
	generator, ok := a.generator.(ToolCallingGenerator)
	if !ok {
		return nil, false, nil
	}
	tools := a.toolRunner.Tools()
	if len(tools) == 0 {
		return nil, false, nil
	}

	messages := []ChatMessage{
		{Role: ChatRoleSystem, Content: BuildToolDecisionPrompt(a.now(), a.location)},
		{Role: ChatRoleUser, Content: question},
	}
	decision, err := generator.GenerateWithTools(ctx, messages, tools)
	if err != nil {
		return nil, true, fmt.Errorf("generate tool decision: %w", err)
	}
	if len(decision.ToolCalls) == 0 {
		content := strings.TrimSpace(decision.Content)
		if content == "" || strings.EqualFold(content, "NO_TOOL") {
			return nil, false, nil
		}
		return &Answer{
			Question: question,
			Answer:   content,
			Model:    a.generator.Model(),
		}, true, nil
	}

	call := decision.ToolCalls[0]
	result, err := a.toolRunner.Execute(ctx, call)
	if err != nil {
		return nil, true, fmt.Errorf("execute tool %s: %w", call.Name, err)
	}
	finalMessages := append([]ChatMessage{}, messages...)
	finalMessages = append(finalMessages, ChatMessage{
		Role:      ChatRoleAssistant,
		Content:   decision.Content,
		ToolCalls: []ChatToolCall{call},
	})
	finalMessages = append(finalMessages, ChatMessage{
		Role:       ChatRoleTool,
		ToolCallID: call.ID,
		Content:    result.Content,
	})
	finalResponse, err := generator.GenerateWithTools(ctx, finalMessages, nil)
	if err != nil {
		return nil, true, fmt.Errorf("generate tool answer: %w", err)
	}
	answer := strings.TrimSpace(finalResponse.Content)
	if answer == "" {
		answer = fallbackToolAnswer(result.Artifacts)
	}
	return &Answer{
		Question:  question,
		Answer:    answer,
		Model:     a.generator.Model(),
		Artifacts: result.Artifacts,
	}, true, nil
}

func BuildToolDecisionPrompt(now time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	var b strings.Builder
	b.WriteString("You route AMA questions about user activity history.\n")
	b.WriteString("Current local time: ")
	b.WriteString(localNow.Format(time.RFC3339))
	b.WriteString("\nTimezone: ")
	b.WriteString(loc.String())
	b.WriteString("\nUse get_usage_timeline only when the user asks about what they were doing: the sessions, transitions, or time spent in a period.\n")
	b.WriteString("Before calling it, infer exact RFC3339 started_at and ended_at values from the question and current local time.\n")
	b.WriteString("Use limit 50 unless the user explicitly asks for another number.\n")
	b.WriteString("Set include_idle to false unless the user explicitly asks about idle time.\n")
	b.WriteString("If this is not a usage question, respond exactly: NO_TOOL\n")
	b.WriteString("After a tool result, answer briefly and mention that the timeline lists the sessions.")
	return b.String()
}

func looksLikeAppUsageQuestion(question string) bool {
	return isAppUsageChartQuestion(question)
}

func isAppUsageChartQuestion(question string) bool {
	normalized := normalizedQuestion(question)
	if !applicationWordRegexp.MatchString(normalized) {
		return false
	}
	return strings.Contains(normalized, "most used") ||
		strings.Contains(normalized, "used most") ||
		strings.Contains(normalized, "use most") ||
		strings.Contains(normalized, "use the most") ||
		strings.Contains(normalized, "used the most") ||
		strings.Contains(normalized, "top app") ||
		strings.Contains(normalized, "top application") ||
		strings.Contains(normalized, "time by app") ||
		strings.Contains(normalized, "time by application") ||
		strings.Contains(normalized, "by app") ||
		strings.Contains(normalized, "by application") ||
		strings.Contains(normalized, "spent time") ||
		strings.Contains(normalized, "spend time") ||
		strings.Contains(normalized, "usage")
}

func normalizedQuestion(question string) string {
	replacer := strings.NewReplacer(
		"?", " ",
		"!", " ",
		",", " ",
		".", " ",
		":", " ",
		";", " ",
		"_", " ",
		"/", " ",
	)
	return " " + strings.Join(strings.Fields(replacer.Replace(strings.ToLower(question))), " ") + " "
}

func (a *Answerer) now() time.Time {
	if a.clock == nil {
		return time.Now()
	}
	return a.clock()
}

func fallbackToolAnswer(artifacts []Artifact) string {
	for _, artifact := range artifacts {
		if artifact.Type != ArtifactTypeUsageTimeline || artifact.UsageTimeline == nil {
			continue
		}
		if artifact.UsageTimeline.TotalEventCount == 0 {
			return "I did not find any usage events in that period."
		}
		return "The timeline lists the usage events for that period."
	}
	return "I found the requested usage timeline."
}

func structuredQuestionLabel(query StructuredQuery) string {
	label := strings.TrimSpace(query.PeriodLabel)
	if label == "" {
		label = "selected period"
	}
	switch query.Kind {
	case StructuredQueryKindTimeline:
		return "Timeline for " + label
	default:
		return "Usage insight for " + label
	}
}

func applyStructuredLabels(artifacts []Artifact, query StructuredQuery) {
	for i := range artifacts {
		switch artifacts[i].Type {
		case ArtifactTypeUsageTimeline:
			if artifacts[i].UsageTimeline != nil && artifacts[i].UsageTimeline.PeriodLabel == "" {
				artifacts[i].UsageTimeline.PeriodLabel = query.PeriodLabel
			}
		}
	}
}

func structuredUsageAnswer(artifacts []Artifact, query StructuredQuery) string {
	for _, artifact := range artifacts {
		switch artifact.Type {
		case ArtifactTypeUsageTimeline:
			if artifact.UsageTimeline == nil {
				continue
			}
			timeline := artifact.UsageTimeline
			label := fallbackPeriodLabel(timeline.PeriodLabel)
			if timeline.TotalEventCount == 0 {
				return fmt.Sprintf("I did not find any usage events for %s.", label)
			}
			if timeline.Truncated {
				return fmt.Sprintf("I found %d usage events for %s and listed the first %d in the timeline.", timeline.TotalEventCount, label, len(timeline.Events))
			}
			return fmt.Sprintf("I found %d usage events for %s. The timeline lists the exact sessions.", timeline.TotalEventCount, label)
		}
	}
	return "I found the requested usage insight."
}

func fallbackPeriodLabel(label string) string {
	if trimmed := strings.TrimSpace(label); trimmed != "" {
		return trimmed
	}
	return "the selected period"
}

func formatAnswerDuration(seconds int64) string {
	if seconds <= 0 {
		return "0 minutes"
	}
	minutes := (seconds + 30) / 60
	if minutes <= 0 {
		minutes = 1
	}
	hours := minutes / 60
	remainder := minutes % 60
	switch {
	case hours == 0:
		return fmt.Sprintf("%d minutes", remainder)
	case remainder == 0:
		return fmt.Sprintf("%d hours", hours)
	default:
		return fmt.Sprintf("%d hours %d minutes", hours, remainder)
	}
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
