package timesheets

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
)

func (s *Server) entryToProto(ctx context.Context, entry queries.TimesheetEntry) (*timesheetsv1.TimesheetEntry, error) {
	usageIDs, err := s.querier.ListTimesheetEntryUsageBlocks(ctx, entry.ID)
	if err != nil {
		return nil, err
	}
	return entryToProtoWithUsageIDs(entry, usageIDs), nil
}

func entryToProtoWithUsageIDs(entry queries.TimesheetEntry, usageIDs []string) *timesheetsv1.TimesheetEntry {
	projectID := ""
	if entry.ProjectID.Valid {
		projectID = entry.ProjectID.String
	}
	return &timesheetsv1.TimesheetEntry{
		Id:              entry.ID,
		ProjectId:       projectID,
		Title:           entry.Title,
		Notes:           entry.Notes,
		StartMinute:     int32(entry.StartMinute),
		DurationMinutes: int32(entry.DurationMinutes),
		Billable:        entry.Billable,
		SourceUsageIds:  usageIDs,
	}
}

func entriesToProto(ctx context.Context, server *Server, rows []queries.TimesheetEntry) ([]*timesheetsv1.TimesheetEntry, error) {
	out := make([]*timesheetsv1.TimesheetEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := server.entryToProto(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}
