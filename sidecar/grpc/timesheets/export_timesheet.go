package timesheets

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ExportTimesheet(ctx context.Context, req *timesheetsv1.ExportTimesheetRequest) (*timesheetsv1.ExportTimesheetResponse, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "timesheet store is unavailable")
	}
	dayStartedAt, dayEndedAt, ok := dayWindow(req.GetDayStartedAt(), req.GetDayEndedAt())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "timesheet day window is invalid")
	}
	entries, err := s.querier.ListTimesheetEntries(ctx, queries.ListTimesheetEntriesParams{
		StartedAt: dayStartedAt,
		EndedAt:   dayEndedAt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list export entries: %v", err)
	}

	switch req.GetFormat() {
	case timesheetsv1.TimesheetExportFormat_TIMESHEET_EXPORT_FORMAT_JSON:
		content, err := exportTimesheetJSON(dayStartedAt, entries)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "export timesheet json: %v", err)
		}
		return &timesheetsv1.ExportTimesheetResponse{
			FileName:    timesheetExportFileName(dayStartedAt, "json"),
			ContentType: "application/json",
			Content:     content,
		}, nil
	case timesheetsv1.TimesheetExportFormat_TIMESHEET_EXPORT_FORMAT_CSV:
		content, err := exportTimesheetCSV(dayStartedAt, entries)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "export timesheet csv: %v", err)
		}
		return &timesheetsv1.ExportTimesheetResponse{
			FileName:    timesheetExportFileName(dayStartedAt, "csv"),
			ContentType: "text/csv",
			Content:     content,
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "timesheet export format is required")
	}
}

type exportedTimesheet struct {
	ExportedAt    time.Time                `json:"exportedAt"`
	Day           time.Time                `json:"day"`
	TotalMs       int64                    `json:"totalMs"`
	BillableMs    int64                    `json:"billableMs"`
	NonBillableMs int64                    `json:"nonBillableMs"`
	Entries       []exportedTimesheetEntry `json:"entries"`
}

type exportedTimesheetEntry struct {
	Title      string    `json:"title"`
	Notes      string    `json:"notes"`
	StartedAt  time.Time `json:"startedAt"`
	DurationMs int64     `json:"durationMs"`
	Billable   bool      `json:"billable"`
}

func exportTimesheetJSON(dayStartedAt time.Time, entries []queries.TimesheetEntry) ([]byte, error) {
	out := exportedTimesheet{
		ExportedAt: time.Now().UTC(),
		Day:        dayStartedAt,
		Entries:    make([]exportedTimesheetEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		durationMs := durationMillis(entry.DurationMinutes)
		out.TotalMs += durationMs
		if entry.Billable {
			out.BillableMs += durationMs
		} else {
			out.NonBillableMs += durationMs
		}
		out.Entries = append(out.Entries, exportedTimesheetEntry{
			Title:      entry.Title,
			Notes:      entry.Notes,
			StartedAt:  startedAtForEntry(dayStartedAt, entry.StartMinute),
			DurationMs: durationMs,
			Billable:   entry.Billable,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}

func exportTimesheetCSV(dayStartedAt time.Time, entries []queries.TimesheetEntry) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"Title", "Notes", "Started At", "Duration (ms)", "Billable"}); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := writer.Write([]string{
			entry.Title,
			entry.Notes,
			startedAtForEntry(dayStartedAt, entry.StartMinute).Format(time.RFC3339),
			strconv.FormatInt(durationMillis(entry.DurationMinutes), 10),
			strconv.FormatBool(entry.Billable),
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func timesheetExportFileName(dayStartedAt time.Time, extension string) string {
	return fmt.Sprintf("timesheet-%s.%s", dayStartedAt.Format("2006-01-02"), extension)
}
