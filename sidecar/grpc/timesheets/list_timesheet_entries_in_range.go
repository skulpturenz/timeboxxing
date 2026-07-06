package timesheets

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListTimesheetEntriesInRange(ctx context.Context, req *timesheetsv1.ListTimesheetEntriesInRangeRequest) (*timesheetsv1.ListTimesheetEntriesInRangeResponse, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	rangeStartedAt, rangeEndedAt, ok := dayWindow(req.GetRangeStartedAt(), req.GetRangeEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet range window is invalid")
	}

	rows, err := s.querier.ListTimesheetEntriesInRange(ctx, queries.ListTimesheetEntriesInRangeParams{
		StartedAt:   rangeStartedAt,
		StartedAt_2: rangeEndedAt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list timesheet entries in range: %v", err)
	}

	// Rows arrive ordered by day started_at, so entries for the same day are contiguous.
	days := make([]*timesheetsv1.RangedTimesheetDay, 0)
	var current *timesheetsv1.RangedTimesheetDay
	for _, row := range rows {
		if current == nil || !row.StartedAt.Equal(current.GetDayStartedAt().AsTime()) {
			current = &timesheetsv1.RangedTimesheetDay{
				DayStartedAt: timestamppb.New(row.StartedAt),
				DayEndedAt:   timestamppb.New(row.EndedAt),
				Entries:      make([]*timesheetsv1.TimesheetEntry, 0),
			}
			days = append(days, current)
		}
		entry, err := s.entryToProto(ctx, rangeRowToEntry(row))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list timesheet entry usage: %v", err)
		}
		current.Entries = append(current.Entries, entry)
	}

	return &timesheetsv1.ListTimesheetEntriesInRangeResponse{Days: days}, nil
}

func rangeRowToEntry(row queries.ListTimesheetEntriesInRangeRow) queries.TimesheetEntry {
	return queries.TimesheetEntry{
		ID:              row.ID,
		TimesheetID:     row.TimesheetID,
		ProjectID:       row.ProjectID,
		Title:           row.Title,
		Notes:           row.Notes,
		StartMinute:     row.StartMinute,
		DurationMinutes: row.DurationMinutes,
		Billable:        row.Billable,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
