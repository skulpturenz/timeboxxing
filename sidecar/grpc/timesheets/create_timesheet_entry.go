package timesheets

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTimesheetEntry(ctx context.Context, req *timesheetsv1.CreateTimesheetEntryRequest) (*timesheetsv1.TimesheetEntry, error) {
	if s.database == nil || s.database.WriteConn == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	dayStartedAt, dayEndedAt, ok := dayWindow(req.GetDayStartedAt(), req.GetDayEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet day window is invalid")
	}
	if !validEntryTime(req.GetStartMinute(), req.GetDurationMinutes()) {
		return nil, status.Error(codes.InvalidArgument, "timesheet entry time is invalid")
	}

	tx, err := s.database.WriteConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin timesheet transaction: %v", err)
	}
	q := queries.New(tx)
	defer tx.Rollback()

	projectID := projectIDOrEmpty(req.GetProjectId())
	projectIDParam := sql.NullString{}
	if projectID != "" {
		count, err := q.CountProjectsByID(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "check project: %v", err)
		}
		if count == 0 {
			return nil, status.Error(codes.InvalidArgument, "project does not exist")
		}
		projectIDParam = sql.NullString{String: projectID, Valid: true}
	}

	timesheet, err := ensureTimesheet(ctx, q, dayStartedAt, dayEndedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert timesheet: %v", err)
	}
	entryID, err := randomID("entry")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate entry id: %v", err)
	}
	now := time.Now().UTC()
	entry, err := q.CreateTimesheetEntry(ctx, queries.CreateTimesheetEntryParams{
		ID:              entryID,
		TimesheetID:     timesheet.ID,
		ProjectID:       projectIDParam,
		Title:           entryTitle(req.GetTitle()),
		Notes:           req.GetNotes(),
		StartMinute:     int64(req.GetStartMinute()),
		DurationMinutes: int64(req.GetDurationMinutes()),
		Billable:        req.GetBillable(),
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create timesheet entry: %v", err)
	}

	usageIDs := sanitizedUsageIDs(req.GetSourceUsageIds())
	for index, usageID := range usageIDs {
		if err := q.CreateTimesheetEntryUsageBlock(ctx, queries.CreateTimesheetEntryUsageBlockParams{
			TimesheetEntryID: entry.ID,
			UsageID:          usageID,
			SortOrder:        int64(index),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "link usage block: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit timesheet entry: %v", err)
	}
	return entryToProtoWithUsageIDs(entry, usageIDs), nil
}

func ensureTimesheet(ctx context.Context, q *queries.Queries, dayStartedAt time.Time, dayEndedAt time.Time) (queries.Timesheet, error) {
	timesheet, err := q.GetTimesheetByWindow(ctx, queries.GetTimesheetByWindowParams{
		StartedAt: dayStartedAt,
		EndedAt:   dayEndedAt,
	})
	if err == nil {
		return timesheet, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return queries.Timesheet{}, err
	}

	id, err := randomID("timesheet")
	if err != nil {
		return queries.Timesheet{}, err
	}
	now := time.Now().UTC()
	return q.CreateTimesheet(ctx, queries.CreateTimesheetParams{
		ID:        id,
		StartedAt: dayStartedAt,
		EndedAt:   dayEndedAt,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func sanitizedUsageIDs(input []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(input))
	for _, usageID := range input {
		id := projectIDOrEmpty(usageID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
