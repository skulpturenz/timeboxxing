package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	appUsageToolNameForAnswerer = "get_app_usage_totals"
	defaultAppUsageChartLimit   = 5
)

var (
	explicitDatePattern   = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	recentWindowPattern   = regexp.MustCompile(`\b(?:last|past)\s+(\d{1,3})\s+(hour|hours|day|days)\b`)
	topLimitPattern       = regexp.MustCompile(`\btop\s+(\d{1,2})\b`)
	applicationWordRegexp = regexp.MustCompile(`\b(app|apps|application|applications|program|programs|software)\b`)
)

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

	if chartAnswer, handled, err := a.answerAppUsageChart(ctx, question); handled || err != nil {
		return chartAnswer, err
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

type appUsageChartRoute struct {
	Recognized    bool
	NeedsPeriod   bool
	StartedAt     time.Time
	EndedAt       time.Time
	PeriodLabel   string
	Limit         int
	IncludeIdle   bool
	Clarification string
}

type appUsageToolArgs struct {
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at"`
	Limit       int    `json:"limit"`
	IncludeIdle bool   `json:"include_idle"`
}

func (a *Answerer) answerAppUsageChart(ctx context.Context, question string) (*Answer, bool, error) {
	route := resolveAppUsageChartRoute(question, a.now(), a.location)
	if !route.Recognized {
		return nil, false, nil
	}
	if route.NeedsPeriod {
		return &Answer{
			Question: question,
			Answer:   route.Clarification,
			Model:    a.generator.Model(),
		}, true, nil
	}
	if a.toolRunner == nil {
		return &Answer{
			Question: question,
			Answer:   "I can chart app usage once the usage history tool is available.",
			Model:    a.generator.Model(),
		}, true, nil
	}

	args, err := json.Marshal(appUsageToolArgs{
		StartedAt:   route.StartedAt.Format(time.RFC3339),
		EndedAt:     route.EndedAt.Format(time.RFC3339),
		Limit:       route.Limit,
		IncludeIdle: route.IncludeIdle,
	})
	if err != nil {
		return nil, true, fmt.Errorf("marshal app usage tool arguments: %w", err)
	}
	result, err := a.toolRunner.Execute(ctx, ChatToolCall{
		ID:        "app_usage_chart",
		Name:      appUsageToolNameForAnswerer,
		Arguments: string(args),
	})
	if err != nil {
		return nil, true, fmt.Errorf("execute app usage chart tool: %w", err)
	}
	for i := range result.Artifacts {
		if result.Artifacts[i].AppUsageChart != nil {
			result.Artifacts[i].AppUsageChart.PeriodLabel = route.PeriodLabel
		}
	}

	return &Answer{
		Question:  question,
		Answer:    appUsageChartAnswer(result.Artifacts, route.PeriodLabel),
		Model:     a.generator.Model(),
		Artifacts: result.Artifacts,
	}, true, nil
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
	b.WriteString("\nUse get_app_usage_totals only when the user asks for most-used apps, top apps, or total time spent per app.\n")
	b.WriteString("Before calling it, infer exact RFC3339 started_at and ended_at values from the question and current local time.\n")
	b.WriteString("Use limit 5 unless the user explicitly asks for another number.\n")
	b.WriteString("Set include_idle to false unless the user explicitly asks about idle time.\n")
	b.WriteString("If the user asks for app usage totals but the time period is unclear, ask one concise clarifying question.\n")
	b.WriteString("If this is not an app usage totals question, respond exactly: NO_TOOL\n")
	b.WriteString("After a tool result, answer briefly and mention that the chart shows the app totals.")
	return b.String()
}

func looksLikeAppUsageQuestion(question string) bool {
	return isAppUsageChartQuestion(question)
}

func resolveAppUsageChartRoute(question string, now time.Time, loc *time.Location) appUsageChartRoute {
	if loc == nil {
		loc = time.Local
	}
	if !isAppUsageChartQuestion(question) {
		return appUsageChartRoute{}
	}
	startedAt, endedAt, label, ok := resolveAppUsagePeriod(question, now, loc)
	if !ok {
		return appUsageChartRoute{
			Recognized:    true,
			NeedsPeriod:   true,
			Limit:         appUsageChartLimit(question),
			IncludeIdle:   appUsageIncludesIdle(question),
			Clarification: "Which time period should I chart for app usage?",
		}
	}
	return appUsageChartRoute{
		Recognized:  true,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		PeriodLabel: label,
		Limit:       appUsageChartLimit(question),
		IncludeIdle: appUsageIncludesIdle(question),
	}
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

func resolveAppUsagePeriod(question string, now time.Time, loc *time.Location) (time.Time, time.Time, string, bool) {
	normalized := normalizedQuestion(question)
	localNow := now.In(loc)
	today := startOfLocalDay(localNow, loc)

	if match := recentWindowPattern.FindStringSubmatch(normalized); len(match) == 3 {
		count, err := strconv.Atoi(match[1])
		if err == nil && count > 0 {
			switch match[2] {
			case "hour", "hours":
				return localNow.Add(-time.Duration(count) * time.Hour), localNow, fmt.Sprintf("Past %d hours", count), true
			case "day", "days":
				return localNow.AddDate(0, 0, -count), localNow, fmt.Sprintf("Past %d days", count), true
			}
		}
	}
	if strings.Contains(normalized, "yesterday") {
		start := today.AddDate(0, 0, -1)
		return start, today, "Yesterday", true
	}
	if strings.Contains(normalized, "today") {
		return today, today.AddDate(0, 0, 1), "Today", true
	}
	if strings.Contains(normalized, "last week") {
		thisWeek := startOfLocalWeek(localNow, loc)
		start := thisWeek.AddDate(0, 0, -7)
		return start, thisWeek, "Last week", true
	}
	if strings.Contains(normalized, "this week") {
		start := startOfLocalWeek(localNow, loc)
		return start, start.AddDate(0, 0, 7), "This week", true
	}
	if strings.Contains(normalized, "last month") {
		thisMonth := startOfLocalMonth(localNow, loc)
		start := thisMonth.AddDate(0, -1, 0)
		return start, thisMonth, "Last month", true
	}
	if strings.Contains(normalized, "this month") {
		start := startOfLocalMonth(localNow, loc)
		return start, start.AddDate(0, 1, 0), "This month", true
	}
	if match := explicitDatePattern.FindStringSubmatch(question); len(match) == 4 {
		year, yearErr := strconv.Atoi(match[1])
		month, monthErr := strconv.Atoi(match[2])
		day, dayErr := strconv.Atoi(match[3])
		if yearErr == nil && monthErr == nil && dayErr == nil {
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
			if start.Year() == year && int(start.Month()) == month && start.Day() == day {
				return start, start.AddDate(0, 0, 1), start.Format("January 2, 2006"), true
			}
		}
	}
	return time.Time{}, time.Time{}, "", false
}

func appUsageChartLimit(question string) int {
	normalized := normalizedQuestion(question)
	if match := topLimitPattern.FindStringSubmatch(normalized); len(match) == 2 {
		value, err := strconv.Atoi(match[1])
		if err == nil && value > 0 {
			if value > 20 {
				return 20
			}
			return value
		}
	}
	return defaultAppUsageChartLimit
}

func appUsageIncludesIdle(question string) bool {
	return strings.Contains(normalizedQuestion(question), "idle")
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

func startOfLocalWeek(value time.Time, loc *time.Location) time.Time {
	day := startOfLocalDay(value, loc)
	daysSinceMonday := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -daysSinceMonday)
}

func startOfLocalMonth(value time.Time, loc *time.Location) time.Time {
	local := value.In(loc)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
}

func (a *Answerer) now() time.Time {
	if a.clock == nil {
		return time.Now()
	}
	return a.clock()
}

func fallbackToolAnswer(artifacts []Artifact) string {
	for _, artifact := range artifacts {
		if artifact.Type != ArtifactTypeAppUsageChart || artifact.AppUsageChart == nil {
			continue
		}
		chart := artifact.AppUsageChart
		if len(chart.Buckets) == 0 {
			return "I did not find any app usage in that period."
		}
		top := chart.Buckets[0]
		return fmt.Sprintf("Your most used app was %s. The chart shows the app totals for that period.", top.Name)
	}
	return "I found the requested usage totals."
}

func structuredQuestionLabel(query StructuredQuery) string {
	label := strings.TrimSpace(query.PeriodLabel)
	if label == "" {
		label = "selected period"
	}
	switch query.Kind {
	case StructuredQueryKindAppTotals:
		return "App totals for " + label
	case StructuredQueryKindTimeline:
		return "Timeline for " + label
	case StructuredQueryKindHabits:
		return "Habits for " + label
	case StructuredQueryKindComparePeriods:
		baseline := strings.TrimSpace(query.BaselinePeriodLabel)
		if baseline == "" {
			baseline = "baseline period"
		}
		return fmt.Sprintf("Compare %s to %s", label, baseline)
	default:
		return "Usage insight for " + label
	}
}

func applyStructuredLabels(artifacts []Artifact, query StructuredQuery) {
	for i := range artifacts {
		switch artifacts[i].Type {
		case ArtifactTypeAppUsageChart:
			if artifacts[i].AppUsageChart != nil && artifacts[i].AppUsageChart.PeriodLabel == "" {
				artifacts[i].AppUsageChart.PeriodLabel = query.PeriodLabel
			}
		case ArtifactTypeUsageTimeline:
			if artifacts[i].UsageTimeline != nil && artifacts[i].UsageTimeline.PeriodLabel == "" {
				artifacts[i].UsageTimeline.PeriodLabel = query.PeriodLabel
			}
		case ArtifactTypeUsageHabitSummary:
			if artifacts[i].UsageHabitSummary != nil && artifacts[i].UsageHabitSummary.PeriodLabel == "" {
				artifacts[i].UsageHabitSummary.PeriodLabel = query.PeriodLabel
			}
		case ArtifactTypeUsageComparison:
			if artifacts[i].UsageComparison != nil {
				if artifacts[i].UsageComparison.CurrentPeriodLabel == "" {
					artifacts[i].UsageComparison.CurrentPeriodLabel = query.PeriodLabel
				}
				if artifacts[i].UsageComparison.BaselinePeriodLabel == "" {
					artifacts[i].UsageComparison.BaselinePeriodLabel = query.BaselinePeriodLabel
				}
			}
		}
	}
}

func structuredUsageAnswer(artifacts []Artifact, query StructuredQuery) string {
	for _, artifact := range artifacts {
		switch artifact.Type {
		case ArtifactTypeAppUsageChart:
			return appUsageChartAnswer(artifacts, query.PeriodLabel)
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
		case ArtifactTypeUsageHabitSummary:
			if artifact.UsageHabitSummary == nil {
				continue
			}
			summary := artifact.UsageHabitSummary
			label := fallbackPeriodLabel(summary.PeriodLabel)
			if summary.SessionCount == 0 {
				return fmt.Sprintf("I did not find any usage events for %s.", label)
			}
			return fmt.Sprintf(
				"For %s, I found %s across %d sessions with %d context switches.",
				label,
				formatAnswerDuration(summary.TotalDurationSeconds),
				summary.SessionCount,
				summary.ContextSwitchCount,
			)
		case ArtifactTypeUsageComparison:
			if artifact.UsageComparison == nil {
				continue
			}
			comparison := artifact.UsageComparison
			currentLabel := fallbackPeriodLabel(comparison.CurrentPeriodLabel)
			baselineLabel := fallbackPeriodLabel(comparison.BaselinePeriodLabel)
			delta := comparison.DurationDeltaSeconds
			direction := "unchanged"
			if delta > 0 {
				direction = "up"
			} else if delta < 0 {
				direction = "down"
			}
			return fmt.Sprintf(
				"%s was %s %s versus %s.",
				currentLabel,
				direction,
				formatAnswerDuration(absInt64(delta)),
				baselineLabel,
			)
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

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func appUsageChartAnswer(artifacts []Artifact, periodLabel string) string {
	for _, artifact := range artifacts {
		if artifact.Type != ArtifactTypeAppUsageChart || artifact.AppUsageChart == nil {
			continue
		}
		chart := artifact.AppUsageChart
		label := strings.TrimSpace(periodLabel)
		if label == "" {
			label = "that period"
		}
		if len(chart.Buckets) == 0 || chart.TotalDurationSeconds <= 0 {
			return fmt.Sprintf("I did not find any app usage for %s. The chart is empty.", label)
		}
		top := chart.Buckets[0]
		return fmt.Sprintf(
			"Your most used app for %s was %s (%s). The chart shows your top %d apps by total time.",
			label,
			top.Name,
			formatAnswerDuration(top.DurationSeconds),
			len(chart.Buckets),
		)
	}
	return "I found the requested app usage totals. The chart shows your app usage by total time."
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
