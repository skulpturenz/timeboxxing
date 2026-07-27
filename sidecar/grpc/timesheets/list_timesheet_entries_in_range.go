package timesheets

import (
	"context"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListTimesheetEntriesInRange(ctx context.Context, req *timesheetsv1.ListTimesheetEntriesInRangeRequest) (*timesheetsv1.ListTimesheetEntriesInRangeResponse, error) {
	if s.readQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	rangeStartedAt, rangeEndedAt, ok := dayWindow(req.GetRangeStartedAt(), req.GetRangeEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet range window is invalid")
	}

	rows, err := s.readQuerier.ListTimesheetEntriesInRange(ctx, readqueries.ListTimesheetEntriesInRangeParams{
		StartedAtUtc:   &rangeStartedAt,
		StartedAtUtc_2: &rangeEndedAt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list timesheet entries in range: %v", err)
	}

	// Rows arrive ordered by started_at_utc, so entries for the same day are contiguous. The day
	// window is derived from the entry timestamp (the ledger no longer stores an explicit day).
	days := make([]*timesheetsv1.RangedTimesheetDay, 0)
	var current *timesheetsv1.RangedTimesheetDay
	for _, row := range rows {
		dayStart := utcDayStart(row.StartedAtUtc)
		if current == nil || !dayStart.Equal(current.GetDayStartedAt().AsTime()) {
			current = &timesheetsv1.RangedTimesheetDay{
				DayStartedAt: timestamppb.New(dayStart),
				DayEndedAt:   timestamppb.New(dayStart.AddDate(0, 0, 1)),
				Entries:      make([]*timesheetsv1.TimesheetEntry, 0),
			}
			days = append(days, current)
		}
		entry, err := s.entryToProto(ctx, dayStart, rangeRowToRow(row))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list timesheet entry usage: %v", err)
		}
		current.Entries = append(current.Entries, entry)
	}

	return &timesheetsv1.ListTimesheetEntriesInRangeResponse{Days: days}, nil
}

func rangeRowToRow(row readqueries.ListTimesheetEntriesInRangeRow) readqueries.ListTimesheetEntriesRow {
	return readqueries.ListTimesheetEntriesRow{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		Title:        row.Title,
		Notes:        row.Notes,
		Billable:     row.Billable,
		StartedAtUtc: row.StartedAtUtc,
		EndedAtUtc:   row.EndedAtUtc,
	}
}
