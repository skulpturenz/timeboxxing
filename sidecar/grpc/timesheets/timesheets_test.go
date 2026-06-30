package timesheets

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateEntryCreatesDailyTimesheetAndListsUsageBlocks(t *testing.T) {
	ctx := context.Background()
	server, database, cleanup := newTestTimesheetsServer(t, ctx)
	defer cleanup()
	dayStart, dayEnd := testDay()

	entry, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		Title:           "  Design review  ",
		Notes:           "Checked wireframes",
		StartMinute:     9 * 60,
		DurationMinutes: 45,
		Billable:        true,
		SourceUsageIds:  []string{"sidecar-2", "sidecar-1", "sidecar-2"},
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if entry.GetProjectId() != "" {
		t.Fatalf("expected no project id, got %q", entry.GetProjectId())
	}
	if entry.GetTitle() != "Design review" {
		t.Fatalf("expected trimmed title, got %q", entry.GetTitle())
	}
	if got := strings.Join(entry.GetSourceUsageIds(), ","); got != "sidecar-2,sidecar-1" {
		t.Fatalf("expected ordered de-duplicated usage ids, got %q", got)
	}

	listed, err := server.ListTimesheetEntries(ctx, &timesheetsv1.ListTimesheetEntriesRequest{
		DayStartedAt: timestamppb.New(dayStart),
		DayEndedAt:   timestamppb.New(dayEnd),
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(listed.GetEntries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(listed.GetEntries()))
	}
	if listed.GetEntries()[0].GetId() != entry.GetId() {
		t.Fatalf("expected listed entry %q, got %q", entry.GetId(), listed.GetEntries()[0].GetId())
	}

	if _, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		Title:           "Follow-up",
		StartMinute:     10 * 60,
		DurationMinutes: 15,
		Billable:        false,
	}); err != nil {
		t.Fatalf("create second entry: %v", err)
	}
	var timesheetCount int
	if err := database.WriteConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM timesheets").Scan(&timesheetCount); err != nil {
		t.Fatalf("count timesheets: %v", err)
	}
	if timesheetCount != 1 {
		t.Fatalf("expected one upserted timesheet, got %d", timesheetCount)
	}
}

func TestCreateEntryValidatesProjectAndTime(t *testing.T) {
	ctx := context.Background()
	server, _, cleanup := newTestTimesheetsServer(t, ctx)
	defer cleanup()
	dayStart, dayEnd := testDay()

	if _, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		ProjectId:       "missing-project",
		Title:           "Work",
		StartMinute:     60,
		DurationMinutes: 30,
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid project, got %v", err)
	}
	if _, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		Title:           "Work",
		StartMinute:     1440,
		DurationMinutes: 30,
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid time, got %v", err)
	}
}

func TestDeletingProjectUnlinksPersistedEntry(t *testing.T) {
	ctx := context.Background()
	server, database, cleanup := newTestTimesheetsServer(t, ctx)
	defer cleanup()
	dayStart, dayEnd := testDay()
	projectID := createTestProject(t, ctx, database)

	entry, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		ProjectId:       projectID,
		Title:           "Project work",
		StartMinute:     60,
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if entry.GetProjectId() != projectID {
		t.Fatalf("expected linked project %q, got %q", projectID, entry.GetProjectId())
	}
	if err := database.WriteQuerier.DeleteProject(ctx, projectID); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	listed, err := server.ListTimesheetEntries(ctx, &timesheetsv1.ListTimesheetEntriesRequest{
		DayStartedAt: timestamppb.New(dayStart),
		DayEndedAt:   timestamppb.New(dayEnd),
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if got := listed.GetEntries()[0].GetProjectId(); got != "" {
		t.Fatalf("expected unlinked project, got %q", got)
	}
}

func TestExportTimesheetJSONAndCSV(t *testing.T) {
	ctx := context.Background()
	server, _, cleanup := newTestTimesheetsServer(t, ctx)
	defer cleanup()
	dayStart, dayEnd := testDay()

	if _, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		Title:           "Design review",
		Notes:           "Line 1\nLine 2",
		StartMinute:     9 * 60,
		DurationMinutes: 45,
		Billable:        true,
	}); err != nil {
		t.Fatalf("create billable entry: %v", err)
	}
	if _, err := server.CreateTimesheetEntry(ctx, &timesheetsv1.CreateTimesheetEntryRequest{
		DayStartedAt:    timestamppb.New(dayStart),
		DayEndedAt:      timestamppb.New(dayEnd),
		Title:           "Admin",
		StartMinute:     10 * 60,
		DurationMinutes: 15,
		Billable:        false,
	}); err != nil {
		t.Fatalf("create non-billable entry: %v", err)
	}

	jsonExport, err := server.ExportTimesheet(ctx, &timesheetsv1.ExportTimesheetRequest{
		DayStartedAt: timestamppb.New(dayStart),
		DayEndedAt:   timestamppb.New(dayEnd),
		Format:       timesheetsv1.TimesheetExportFormat_TIMESHEET_EXPORT_FORMAT_JSON,
	})
	if err != nil {
		t.Fatalf("export json: %v", err)
	}
	if jsonExport.GetFileName() != "timesheet-2025-05-01.json" {
		t.Fatalf("unexpected json filename %q", jsonExport.GetFileName())
	}
	var payload struct {
		TotalMs       int64 `json:"totalMs"`
		BillableMs    int64 `json:"billableMs"`
		NonBillableMs int64 `json:"nonBillableMs"`
		Entries       []struct {
			Title      string `json:"title"`
			DurationMs int64  `json:"durationMs"`
			Billable   bool   `json:"billable"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(jsonExport.GetContent(), &payload); err != nil {
		t.Fatalf("parse json export: %v", err)
	}
	if payload.TotalMs != 60*60_000 || payload.BillableMs != 45*60_000 || payload.NonBillableMs != 15*60_000 {
		t.Fatalf("unexpected totals: %+v", payload)
	}
	if len(payload.Entries) != 2 || payload.Entries[0].Title != "Design review" {
		t.Fatalf("unexpected entries: %+v", payload.Entries)
	}

	csvExport, err := server.ExportTimesheet(ctx, &timesheetsv1.ExportTimesheetRequest{
		DayStartedAt: timestamppb.New(dayStart),
		DayEndedAt:   timestamppb.New(dayEnd),
		Format:       timesheetsv1.TimesheetExportFormat_TIMESHEET_EXPORT_FORMAT_CSV,
	})
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if csvExport.GetFileName() != "timesheet-2025-05-01.csv" {
		t.Fatalf("unexpected csv filename %q", csvExport.GetFileName())
	}
	rows, err := csv.NewReader(strings.NewReader(string(csvExport.GetContent()))).ReadAll()
	if err != nil {
		t.Fatalf("parse csv export: %v", err)
	}
	if got := strings.Join(rows[0], ","); got != "Title,Notes,Started At,Duration (ms),Billable" {
		t.Fatalf("unexpected csv headers %q", got)
	}
	if rows[1][0] != "Design review" || rows[1][3] != "2700000" || rows[1][4] != "true" {
		t.Fatalf("unexpected first csv row: %#v", rows[1])
	}
}

func newTestTimesheetsServer(t *testing.T, ctx context.Context) (*Server, *db.Database, func()) {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		Engine:         db.EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	registry := services.New()
	db.Register(registry, database)
	server := NewServer(registry)

	return server, database, func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	}
}

func createTestProject(t *testing.T, ctx context.Context, database *db.Database) string {
	t.Helper()

	now := time.Now().UTC()
	project, err := database.WriteQuerier.CreateProject(ctx, queries.CreateProjectParams{
		ID:        "client-work",
		Name:      "Client Work",
		ColorArgb: 0xFF00FFEE,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project.ID
}

func testDay() (time.Time, time.Time) {
	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	return start, start.Add(24 * time.Hour)
}
