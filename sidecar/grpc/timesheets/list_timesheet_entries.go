package timesheets

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListTimesheetEntries(ctx context.Context, req *timesheetsv1.ListTimesheetEntriesRequest) (*timesheetsv1.ListTimesheetEntriesResponse, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	dayStartedAt, dayEndedAt, ok := dayWindow(req.GetDayStartedAt(), req.GetDayEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet day window is invalid")
	}

	rows, err := s.querier.ListTimesheetEntries(ctx, queries.ListTimesheetEntriesParams{
		StartedAt: dayStartedAt,
		EndedAt:   dayEndedAt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list timesheet entries: %v", err)
	}
	entries, err := entriesToProto(ctx, s, rows)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list timesheet entry usage: %v", err)
	}
	return &timesheetsv1.ListTimesheetEntriesResponse{Entries: entries}, nil
}
