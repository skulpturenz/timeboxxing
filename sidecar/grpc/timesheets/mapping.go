package timesheets

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
)

// costingTypeHourlyID matches the project_costing_types seed (db/seeds/project_costing_types).
const costingTypeHourlyID = 1

// usageIDPrefix is the wire convention linking a ledger item to a timeline entry ("sidecar-<timeline id>").
const usageIDPrefix = "sidecar-"

func usageIDForTimeline(timelineID int64) string {
	return usageIDPrefix + strconv.FormatInt(timelineID, 10)
}

func parseUsageTimelineID(usageID string) (int64, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(usageID), usageIDPrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func usageIDStrings(timelineIDs []sql.NullInt64) []string {
	out := make([]string, 0, len(timelineIDs))
	for _, id := range timelineIDs {
		if id.Valid {
			out = append(out, usageIDForTimeline(id.Int64))
		}
	}
	return out
}

// entryMinutes converts a ledger item's absolute start/end timestamps to day-relative minutes.
func entryMinutes(dayStart time.Time, startedAt, endedAt sql.NullTime) (int32, int32) {
	var startMinute int32
	if startedAt.Valid {
		startMinute = int32(startedAt.Time.UTC().Sub(dayStart).Minutes())
	}
	var durationMinutes int32
	if startedAt.Valid && endedAt.Valid {
		durationMinutes = int32(endedAt.Time.Sub(startedAt.Time).Minutes())
	}
	return startMinute, durationMinutes
}

func buildTimesheetEntryProto(dayStart time.Time, id int64, projectID sql.NullInt64, title string, notes sql.NullString, billable bool, startedAtUtc, endedAtUtc sql.NullTime, usageIDs []string) *timesheetsv1.TimesheetEntry {
	startMinute, durationMinutes := entryMinutes(dayStart, startedAtUtc, endedAtUtc)
	entry := &timesheetsv1.TimesheetEntry{
		Id:              id,
		Title:           title,
		Notes:           notes.String,
		StartMinute:     startMinute,
		DurationMinutes: durationMinutes,
		Billable:        billable,
		SourceUsageIds:  usageIDs,
	}
	if projectID.Valid {
		entry.ProjectId = projectID.Int64
	}
	return entry
}

func (s *Server) entryToProto(ctx context.Context, dayStart time.Time, entry readqueries.ListTimesheetEntriesRow) (*timesheetsv1.TimesheetEntry, error) {
	usageIDs, err := s.readQuerier.ListTimesheetEntryUsageBlocks(ctx, sql.NullInt64{Int64: entry.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	return buildTimesheetEntryProto(dayStart, entry.ID, entry.ProjectID, entry.Title, entry.Notes, entry.Billable, entry.StartedAtUtc, entry.EndedAtUtc, usageIDStrings(usageIDs)), nil
}

func entriesToProto(ctx context.Context, server *Server, dayStart time.Time, rows []readqueries.ListTimesheetEntriesRow) ([]*timesheetsv1.TimesheetEntry, error) {
	out := make([]*timesheetsv1.TimesheetEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := server.entryToProto(ctx, dayStart, row)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}

// utcDayStart returns UTC midnight of the given timestamp (used to group range results by day, since
// the ledger no longer stores an explicit day window).
func utcDayStart(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	u := t.Time.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
