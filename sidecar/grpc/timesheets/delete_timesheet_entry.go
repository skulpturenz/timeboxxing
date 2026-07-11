package timesheets

import (
	"context"

	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) DeleteTimesheetEntry(ctx context.Context, req *timesheetsv1.DeleteTimesheetEntryRequest) (*timesheetsv1.DeleteTimesheetEntryResponse, error) {
	if s.writeQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "timesheet entry id is required")
	}
	if err := s.writeQuerier.DeleteLedgerItem(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete timesheet entry: %v", err)
	}
	return &timesheetsv1.DeleteTimesheetEntryResponse{}, nil
}
