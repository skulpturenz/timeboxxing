package timesheets

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mattn/go-sqlite3"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTimesheetEntry(ctx context.Context, req *timesheetsv1.CreateTimesheetEntryRequest) (*timesheetsv1.TimesheetEntry, error) {
	if s.database == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	dayStartedAt, dayEndedAt, ok := dayWindow(req.GetDayStartedAt(), req.GetDayEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet day window is invalid")
	}
	if !validEntryTime(req.GetStartMinute(), req.GetDurationMinutes()) {
		return nil, status.Error(codes.InvalidArgument, "timesheet entry time is invalid")
	}

	projectIDParam := sql.NullString{}
	if projectID := projectIDOrEmpty(req.GetProjectId()); projectID != "" {
		projectIDParam = sql.NullString{String: projectID, Valid: true}
	}

	timesheetID, err := randomID("timesheet")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate timesheet id: %v", err)
	}
	entryID, err := randomID("entry")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate entry id: %v", err)
	}
	usageIDs := sanitizedUsageIDs(req.GetSourceUsageIds())

	var entry writequeries.TimesheetEntry
	if err := s.database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
		now := time.Now().UTC()
		// Get-or-create the timesheet for this window in one write (see EnsureTimesheet).
		timesheet, err := q.EnsureTimesheet(ctx, writequeries.EnsureTimesheetParams{
			ID:        timesheetID,
			StartedAt: dayStartedAt,
			EndedAt:   dayEndedAt,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "upsert timesheet: %v", err)
		}

		entry, err = q.CreateTimesheetEntry(ctx, writequeries.CreateTimesheetEntryParams{
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
			// A non-existent project_id trips the timesheet_entries -> projects foreign key.
			if isForeignKeyConstraintErr(err) {
				return status.Error(codes.InvalidArgument, "project does not exist")
			}
			return status.Errorf(codes.Internal, "create timesheet entry: %v", err)
		}

		for index, usageID := range usageIDs {
			if err := q.CreateTimesheetEntryUsageBlock(ctx, writequeries.CreateTimesheetEntryUsageBlockParams{
				TimesheetEntryID: entry.ID,
				UsageID:          usageID,
				SortOrder:        int64(index),
			}); err != nil {
				return status.Errorf(codes.Internal, "link usage block: %v", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return entryToProtoWithUsageIDs(writeEntryToRead(entry), usageIDs), nil
}

func isForeignKeyConstraintErr(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey
}

// writeEntryToRead converts a write-package entry row into the read-package struct the proto mappers
// are typed on (identical fields; the two sqlc packages generate distinct types).
func writeEntryToRead(e writequeries.TimesheetEntry) readqueries.TimesheetEntry {
	return readqueries.TimesheetEntry{
		ID:              e.ID,
		TimesheetID:     e.TimesheetID,
		ProjectID:       e.ProjectID,
		Title:           e.Title,
		Notes:           e.Notes,
		StartMinute:     e.StartMinute,
		DurationMinutes: e.DurationMinutes,
		Billable:        e.Billable,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
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
