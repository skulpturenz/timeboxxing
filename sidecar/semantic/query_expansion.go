package semantic

import (
	"fmt"
	"strings"
	"time"
)

func ExpandSemanticQuery(question string, now time.Time, loc *time.Location) string {
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	normalized := strings.ToLower(trimmed)
	var contextLines []string

	if strings.Contains(normalized, "today") {
		start := startOfLocalDay(localNow, loc)
		contextLines = append(contextLines, dateRangeLine("today", start, start.AddDate(0, 0, 1), loc))
	}
	if strings.Contains(normalized, "yesterday") {
		start := startOfLocalDay(localNow, loc).AddDate(0, 0, -1)
		contextLines = append(contextLines, dateRangeLine("yesterday", start, start.AddDate(0, 0, 1), loc))
	}
	if strings.Contains(normalized, "this morning") || strings.Contains(normalized, "morning") {
		start := startOfLocalDay(localNow, loc).Add(6 * time.Hour)
		contextLines = append(contextLines, dateRangeLine("this morning", start, start.Add(6*time.Hour), loc))
	}
	if strings.Contains(normalized, "this afternoon") || strings.Contains(normalized, "afternoon") {
		start := startOfLocalDay(localNow, loc).Add(12 * time.Hour)
		contextLines = append(contextLines, dateRangeLine("this afternoon", start, start.Add(6*time.Hour), loc))
	}
	if strings.Contains(normalized, "this evening") || strings.Contains(normalized, "evening") {
		start := startOfLocalDay(localNow, loc).Add(18 * time.Hour)
		contextLines = append(contextLines, dateRangeLine("this evening", start, start.Add(6*time.Hour), loc))
	}

	if len(contextLines) == 0 {
		return trimmed
	}

	return fmt.Sprintf(
		"Original question: %s\nLocal semantic query context:\n%s",
		trimmed,
		strings.Join(contextLines, "\n"),
	)
}

func dateRangeLine(label string, start, end time.Time, loc *time.Location) string {
	return fmt.Sprintf(
		"%s means %s from %s to %s local time.",
		label,
		start.In(loc).Format("Monday, January 2, 2006"),
		start.In(loc).Format("3:04 PM"),
		end.In(loc).Format("3:04 PM"),
	)
}
