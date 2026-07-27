package timesheets

import (
	"context"
	"errors"
	"time"

	"github.com/mattn/go-sqlite3"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTimesheetEntry(ctx context.Context, req *timesheetsv1.CreateTimesheetEntryRequest) (*timesheetsv1.TimesheetEntry, error) {
	if s.database == nil || s.database.WriteQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	dayStartedAt, _, ok := dayWindow(req.GetDayStartedAt(), req.GetDayEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet day window is invalid")
	}
	if !validEntryTime(req.GetStartMinute(), req.GetDurationMinutes()) {
		return nil, status.Error(codes.InvalidArgument, "timesheet entry time is invalid")
	}

	startedAt := startedAtForEntry(dayStartedAt, int64(req.GetStartMinute()))
	endedAt := startedAt.Add(time.Duration(req.GetDurationMinutes()) * time.Minute)

	var projectID *int64
	if req.GetProjectId() > 0 {
		id := req.GetProjectId()
		projectID = &id
	}

	// Usage ids arrive as "sidecar-<timeline id>" strings; parse them to timeline ids to link.
	timelineIDs := make([]int64, 0, len(req.GetSourceUsageIds()))
	seen := map[int64]bool{}
	for _, usageID := range req.GetSourceUsageIds() {
		id, ok := parseUsageTimelineID(usageID)
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		timelineIDs = append(timelineIDs, id)
	}

	var entry writequeries.LedgerItem
	if err := s.database.WriteQuerier.WriteTx(ctx, func(q *writequeries.Queries) error {
		// The ledger is not seeded; it is created lazily the first time an entry is recorded.
		if err := q.EnsureLedger(ctx); err != nil {
			return status.Errorf(codes.Internal, "ensure ledger: %v", err)
		}

		var err error
		notes := req.GetNotes()
		entry, err = q.CreateLedgerItem(ctx, writequeries.CreateLedgerItemParams{
			Billable:     req.GetBillable(),
			Title:        entryTitle(req.GetTitle()),
			Notes:        &notes,
			StartedAtUtc: &startedAt,
			EndedAtUtc:   &endedAt,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "create ledger item: %v", err)
		}

		if projectID != nil {
			costingTypeID := int64(costingTypeHourlyID)
			if err := q.CreateProjectCost(ctx, writequeries.CreateProjectCostParams{
				LedgerItemsID: &entry.ID,
				ProjectsID:    projectID,
				CostingTypeID: &costingTypeID,
				Rate:          nil,
			}); err != nil {
				// A non-existent project_id trips the project_costs -> projects foreign key.
				if isForeignKeyConstraintErr(err) {
					return status.Error(codes.InvalidArgument, "project does not exist")
				}
				return status.Errorf(codes.Internal, "link project cost: %v", err)
			}
		}

		for _, timelineID := range timelineIDs {
			if err := q.CreateLedgerItemTimelineEntry(ctx, writequeries.CreateLedgerItemTimelineEntryParams{
				LedgerItemsID: &entry.ID,
				TimelineID:    &timelineID,
			}); err != nil {
				return status.Errorf(codes.Internal, "link timeline entry: %v", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	linkedUsageIDs := make([]string, 0, len(timelineIDs))
	for _, id := range timelineIDs {
		linkedUsageIDs = append(linkedUsageIDs, usageIDForTimeline(id))
	}
	return buildTimesheetEntryProto(dayStartedAt, entry.ID, projectID, entry.Title, entry.Notes, entry.Billable, entry.StartedAtUtc, entry.EndedAtUtc, linkedUsageIDs), nil
}

func isForeignKeyConstraintErr(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey
}
