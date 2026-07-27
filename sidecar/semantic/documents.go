package semantic

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	enumssemanticdocumenttype "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_semantic_document_type"
)

const (
	DocumentTypeEvent        = "event"
	DocumentTypeDaySummary   = "day_summary"
	DocumentTypeAppDay       = "app_day_summary"
	DocumentTypeTimeBlock    = "time_block_summary"
	semanticDocumentDateForm = "2006-01-02"
)

// documentTypeID resolves a document-type code to its semantic_document_types id.
func documentTypeID(code string) *int64 {
	documentType, err := enumssemanticdocumenttype.Parse(code)
	if err != nil {
		return nil
	}
	id := int64(documentType)
	return &id
}

// toNullString / toNullBool adapt the nullable pointer columns emitted by the generated queries back
// into the sql.Null* values the internal document renderers consume.
func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func toNullBool(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}

// deriveEventReason reconstructs the transition reason for a single event from its idle flag and the
// previous timeline entry (reason is no longer stored on the event store).
func deriveEventReason(idle *bool, applicationID, prevApplicationID *int64, prevIdle *bool) string {
	if idle != nil && *idle {
		return "idle"
	}
	if prevIdle != nil && *prevIdle {
		return "return_from_idle"
	}
	if applicationID != nil && prevApplicationID != nil && *applicationID == *prevApplicationID {
		return "tab_change"
	}
	return "focus_change"
}

type DocumentSpec struct {
	Key               string
	Type              string
	TransitionEventID *int64
	StartedAt         sql.NullTime
	EndedAt           sql.NullTime
	Content           string
}

type transitionDocumentSource struct {
	TransitionEventID int64
	ApplicationName   sql.NullString
	Reason            string
	StartedAt         time.Time
	EndedAt           time.Time
	Browser           sql.NullBool
	Tab               *string
	Idle              sql.NullBool
	CdpURL            *string
}

type sourceAggregate struct {
	Name         string
	SourceType   string
	Duration     time.Duration
	SessionCount int
	Titles       map[string]struct{}
	Hosts        map[string]struct{}
}

type timeBlock struct {
	Key       string
	Label     string
	StartHour int
	EndHour   int
}

var semanticTimeBlocks = []timeBlock{
	{Key: "overnight", Label: "overnight", StartHour: 0, EndHour: 6},
	{Key: "morning", Label: "morning", StartHour: 6, EndHour: 12},
	{Key: "afternoon", Label: "afternoon", StartHour: 12, EndHour: 18},
	{Key: "evening", Label: "evening", StartHour: 18, EndHour: 24},
}

func eventDocumentSpec(src readqueries.GetSemanticEventDocumentSourceRow, loc *time.Location) DocumentSpec {
	source := transitionDocumentSource{
		TransitionEventID: src.TransitionEventID,
		ApplicationName:   toNullString(src.ApplicationName),
		Reason:            deriveEventReason(src.Idle, src.ApplicationID, src.PrevApplicationID, src.PrevIdle),
		StartedAt:         src.StartedAt,
		EndedAt:           src.EndedAt,
		Browser:           toNullBool(src.Browser),
		Tab:               src.Tab,
		Idle:              toNullBool(src.Idle),
		CdpURL:            src.CdpUrl,
	}
	transitionEventID := src.TransitionEventID
	return DocumentSpec{
		Key:               fmt.Sprintf("event:%d", src.TransitionEventID),
		Type:              DocumentTypeEvent,
		TransitionEventID: &transitionEventID,
		StartedAt:         sql.NullTime{Time: src.StartedAt, Valid: !src.StartedAt.IsZero()},
		EndedAt:           sql.NullTime{Time: src.EndedAt, Valid: !src.EndedAt.IsZero()},
		Content:           renderEventDocument(source, loc),
	}
}

func summaryDocumentSpecs(rows []readqueries.ListTransitionEventDocumentSourcesForWindowRow, day time.Time, loc *time.Location) []DocumentSpec {
	if loc == nil {
		loc = time.Local
	}
	dayStart := startOfLocalDay(day, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	sources := make([]transitionDocumentSource, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, transitionDocumentSource{
			TransitionEventID: row.TransitionEventID,
			ApplicationName:   toNullString(row.ApplicationName),
			// Reason is unused by the summary renderers (only per-event documents surface it).
			StartedAt: row.StartedAt,
			EndedAt:   row.EndedAt,
			Browser:   toNullBool(row.Browser),
			Tab:       row.Tab,
			Idle:      toNullBool(row.Idle),
			CdpURL:    row.CdpUrl,
		})
	}

	dateKey := dayStart.Format(semanticDocumentDateForm)
	specs := []DocumentSpec{{
		Key:       "day:" + dateKey,
		Type:      DocumentTypeDaySummary,
		StartedAt: sql.NullTime{Time: dayStart, Valid: true},
		EndedAt:   sql.NullTime{Time: dayEnd, Valid: true},
		Content:   renderDaySummaryDocument(sources, dayStart, dayEnd, loc),
	}}

	for _, aggregate := range aggregateSources(sources, dayStart, dayEnd) {
		specs = append(specs, DocumentSpec{
			Key:       fmt.Sprintf("app_day:%s:%s", dateKey, stableKeyPart(aggregate.Name)),
			Type:      DocumentTypeAppDay,
			StartedAt: sql.NullTime{Time: dayStart, Valid: true},
			EndedAt:   sql.NullTime{Time: dayEnd, Valid: true},
			Content:   renderAppDaySummaryDocument(aggregate, dayStart, dayEnd, loc),
		})
	}

	for _, block := range semanticTimeBlocks {
		blockStart := dayStart.Add(time.Duration(block.StartHour) * time.Hour)
		blockEnd := dayStart.Add(time.Duration(block.EndHour) * time.Hour)
		specs = append(specs, DocumentSpec{
			Key:       fmt.Sprintf("time_block:%s:%s", dateKey, block.Key),
			Type:      DocumentTypeTimeBlock,
			StartedAt: sql.NullTime{Time: blockStart, Valid: true},
			EndedAt:   sql.NullTime{Time: blockEnd, Valid: true},
			Content:   renderTimeBlockSummaryDocument(sources, block, blockStart, blockEnd, loc),
		})
	}

	return specs
}

func renderEventDocument(src transitionDocumentSource, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	title := eventTitle(src)
	sourceType := eventSourceType(src)
	appName := nullStringValue(src.ApplicationName.Valid, src.ApplicationName.String, "None")
	host := hostFromURL(stringPointerValue(src.CdpURL, ""))
	localStart := src.StartedAt.In(loc)
	localEnd := src.EndedAt.In(loc)

	lines := []string{
		"Document type: event",
		fmt.Sprintf("Transition event id: %d", src.TransitionEventID),
		fmt.Sprintf("Source type: %s", sourceType),
		fmt.Sprintf("Title: %s", title),
		fmt.Sprintf("Application: %s", appName),
		fmt.Sprintf("Local date: %s", localStart.Format("Monday, January 2, 2006")),
		fmt.Sprintf("Local time range: %s to %s", localStart.Format("3:04 PM"), localEnd.Format("3:04 PM")),
		fmt.Sprintf("Duration: %s", formatSemanticDuration(src.EndedAt.Sub(src.StartedAt))),
		fmt.Sprintf("Reason: %s", src.Reason),
		fmt.Sprintf("Browser: %t", src.Browser.Valid && src.Browser.Bool),
		fmt.Sprintf("Tab or window title: %s", stringPointerValue(src.Tab, "None")),
		fmt.Sprintf("URL host: %s", nullStringValue(host != "", host, "None")),
		fmt.Sprintf("Idle: %t", src.Idle.Valid && src.Idle.Bool),
	}
	return strings.Join(lines, "\n")
}

func renderDaySummaryDocument(sources []transitionDocumentSource, dayStart, dayEnd time.Time, loc *time.Location) string {
	total := clippedTotalDuration(sources, dayStart, dayEnd)
	aggregates := aggregateSources(sources, dayStart, dayEnd)
	lines := []string{
		"Document type: day_summary",
		fmt.Sprintf("Local date: %s", dayStart.In(loc).Format("Monday, January 2, 2006")),
		fmt.Sprintf("Local day range: %s to %s", dayStart.In(loc).Format("3:04 PM"), dayEnd.In(loc).Format("3:04 PM")),
		fmt.Sprintf("Total captured usage: %s", formatSemanticDuration(total)),
		fmt.Sprintf("Completed usage sessions: %d", len(sources)),
		"App and source totals:",
	}
	if len(aggregates) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, aggregate := range aggregates {
			lines = append(lines, fmt.Sprintf("- %s (%s): %s across %d sessions", aggregate.Name, aggregate.SourceType, formatSemanticDuration(aggregate.Duration), aggregate.SessionCount))
		}
	}
	lines = append(lines, "Highlights:")
	lines = append(lines, topTitlesLine(aggregates)...)
	return strings.Join(lines, "\n")
}

func renderAppDaySummaryDocument(aggregate sourceAggregate, dayStart, dayEnd time.Time, loc *time.Location) string {
	lines := []string{
		"Document type: app_day_summary",
		fmt.Sprintf("Local date: %s", dayStart.In(loc).Format("Monday, January 2, 2006")),
		fmt.Sprintf("Source: %s", aggregate.Name),
		fmt.Sprintf("Source type: %s", aggregate.SourceType),
		fmt.Sprintf("Total duration: %s", formatSemanticDuration(aggregate.Duration)),
		fmt.Sprintf("Session count: %d", aggregate.SessionCount),
		fmt.Sprintf("Day range: %s to %s", dayStart.In(loc).Format("3:04 PM"), dayEnd.In(loc).Format("3:04 PM")),
		fmt.Sprintf("Titles: %s", joinSetValues(aggregate.Titles, "None", 8)),
		fmt.Sprintf("URL hosts: %s", joinSetValues(aggregate.Hosts, "None", 8)),
	}
	return strings.Join(lines, "\n")
}

func renderTimeBlockSummaryDocument(sources []transitionDocumentSource, block timeBlock, blockStart, blockEnd time.Time, loc *time.Location) string {
	total := clippedTotalDuration(sources, blockStart, blockEnd)
	aggregates := aggregateSources(sources, blockStart, blockEnd)
	lines := []string{
		"Document type: time_block_summary",
		fmt.Sprintf("Local date: %s", blockStart.In(loc).Format("Monday, January 2, 2006")),
		fmt.Sprintf("Time block: %s", block.Label),
		fmt.Sprintf("Local time range: %s to %s", blockStart.In(loc).Format("3:04 PM"), blockEnd.In(loc).Format("3:04 PM")),
		fmt.Sprintf("Total captured usage in block: %s", formatSemanticDuration(total)),
		"App and source totals in block:",
	}
	if len(aggregates) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, aggregate := range aggregates {
			lines = append(lines, fmt.Sprintf("- %s (%s): %s across %d sessions", aggregate.Name, aggregate.SourceType, formatSemanticDuration(aggregate.Duration), aggregate.SessionCount))
		}
	}
	return strings.Join(lines, "\n")
}

func aggregateSources(sources []transitionDocumentSource, windowStart, windowEnd time.Time) []sourceAggregate {
	byName := map[string]*sourceAggregate{}
	for _, source := range sources {
		duration := clippedDuration(source.StartedAt, source.EndedAt, windowStart, windowEnd)
		if duration <= 0 {
			continue
		}
		name := eventSourceName(source)
		aggregate := byName[name]
		if aggregate == nil {
			aggregate = &sourceAggregate{
				Name:       name,
				SourceType: eventSourceType(source),
				Titles:     map[string]struct{}{},
				Hosts:      map[string]struct{}{},
			}
			byName[name] = aggregate
		}
		aggregate.Duration += duration
		aggregate.SessionCount++
		if title := eventTitle(source); title != "" && title != name {
			aggregate.Titles[title] = struct{}{}
		}
		if host := hostFromURL(stringPointerValue(source.CdpURL, "")); host != "" {
			aggregate.Hosts[host] = struct{}{}
		}
	}

	values := make([]sourceAggregate, 0, len(byName))
	for _, aggregate := range byName {
		values = append(values, *aggregate)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Duration == values[j].Duration {
			return values[i].Name < values[j].Name
		}
		return values[i].Duration > values[j].Duration
	})
	return values
}

func topTitlesLine(aggregates []sourceAggregate) []string {
	var lines []string
	for _, aggregate := range aggregates {
		titles := joinSetValues(aggregate.Titles, "", 3)
		if titles == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", aggregate.Name, titles))
		if len(lines) >= 8 {
			break
		}
	}
	if len(lines) == 0 {
		return []string{"- None"}
	}
	return lines
}

func eventSourceName(source transitionDocumentSource) string {
	if source.Idle.Valid && source.Idle.Bool {
		return "Idle"
	}
	if source.ApplicationName.Valid && strings.TrimSpace(source.ApplicationName.String) != "" {
		return strings.TrimSpace(source.ApplicationName.String)
	}
	return "Unknown application"
}

func eventTitle(source transitionDocumentSource) string {
	if source.Idle.Valid && source.Idle.Bool {
		return "Idle"
	}
	if source.Tab != nil && strings.TrimSpace(*source.Tab) != "" {
		return strings.TrimSpace(*source.Tab)
	}
	return eventSourceName(source)
}

func eventSourceType(source transitionDocumentSource) string {
	switch {
	case source.Idle.Valid && source.Idle.Bool:
		return "idle"
	case source.Browser.Valid && source.Browser.Bool:
		return "browser"
	default:
		return "application"
	}
}

func clippedTotalDuration(sources []transitionDocumentSource, windowStart, windowEnd time.Time) time.Duration {
	var total time.Duration
	for _, source := range sources {
		total += clippedDuration(source.StartedAt, source.EndedAt, windowStart, windowEnd)
	}
	return total
}

func clippedDuration(start, end, windowStart, windowEnd time.Time) time.Duration {
	if start.Before(windowStart) {
		start = windowStart
	}
	if end.After(windowEnd) {
		end = windowEnd
	}
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}

func startOfLocalDay(value time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	local := value.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func stableKeyPart(value string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:8])
}

func hostFromURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func nullStringValue(valid bool, value string, fallback string) string {
	if !valid || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func stringPointerValue(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func joinSetValues(values map[string]struct{}, fallback string, limit int) string {
	if len(values) == 0 {
		return fallback
	}
	parts := make([]string, 0, len(values))
	for value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	sort.Strings(parts)
	if limit > 0 && len(parts) > limit {
		parts = parts[:limit]
	}
	if len(parts) == 0 {
		return fallback
	}
	return strings.Join(parts, ", ")
}

func formatSemanticDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0 minutes"
	}
	minutes := int(math.Round(duration.Minutes()))
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
