package reporter

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

func TestDatabaseRecordUpsertsApplicationAndMetadata(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	reporter := NewDatabaseReporter(database.WriteConn)
	assertSeededTransitionReasons(t, ctx, database.WriteConn)

	started := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(5 * time.Minute)

	transition := session.Transition{
		From: &session.Session{
			Key: session.AppKey{
				AppName:  "  Google Chrome  ",
				TabTitle: "GitHub",
				CDPURL:   "https://github.com/",
			},
			StartedAt: started,
			EndedAt:   ended,
			Duration:  ended.Sub(started),
		},
		Reason: session.ReasonTabChange,
	}

	if _, err := reporter.Record(ctx, transition); err != nil {
		t.Fatalf("record first transition: %v", err)
	}
	if _, err := reporter.Record(ctx, transition); err != nil {
		t.Fatalf("record second transition: %v", err)
	}

	var appCount int
	var appName string
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT COUNT(*), MAX(name) FROM applications`).Scan(&appCount, &appName); err != nil {
		t.Fatalf("query applications: %v", err)
	}
	if appCount != 1 {
		t.Fatalf("expected one normalized application, got %d", appCount)
	}
	if appName != "Google Chrome" {
		t.Fatalf("expected trimmed application name, got %q", appName)
	}

	var eventCount int
	var eventsWithApplication int
	var reasonID int64
	if err := database.WriteConn.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(application_id), MAX(transition_reason_id)
		FROM transition_events
	`).Scan(&eventCount, &eventsWithApplication, &reasonID); err != nil {
		t.Fatalf("query transition events: %v", err)
	}
	if eventCount != 2 || eventsWithApplication != 2 {
		t.Fatalf("expected two transition events with applications, got count=%d application_count=%d", eventCount, eventsWithApplication)
	}
	if reasonID != 4 {
		t.Fatalf("expected tab_change reason id 4, got %d", reasonID)
	}

	var browser bool
	var tab sql.NullString
	var idle bool
	var cdpURL sql.NullString
	if err := database.WriteConn.QueryRowContext(ctx, `
		SELECT browser, tab, idle, cdp_url
		FROM transition_event_metadata
		ORDER BY id
		LIMIT 1
	`).Scan(&browser, &tab, &idle, &cdpURL); err != nil {
		t.Fatalf("query transition event metadata: %v", err)
	}
	if !browser {
		t.Fatal("expected browser metadata to be true")
	}
	if !tab.Valid || tab.String != "GitHub" {
		t.Fatalf("expected tab metadata, got valid=%t value=%q", tab.Valid, tab.String)
	}
	if idle {
		t.Fatal("expected idle metadata to be false")
	}
	if !cdpURL.Valid || cdpURL.String != "https://github.com/" {
		t.Fatalf("expected cdp_url metadata, got valid=%t value=%q", cdpURL.Valid, cdpURL.String)
	}
}

func TestDatabaseRecordIdleEventHasNullApplication(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	reporter := NewDatabaseReporter(database.WriteConn)

	started := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(10 * time.Minute)

	if _, err := reporter.Record(ctx, session.Transition{
		From: &session.Session{
			Key:       session.AppKey{IsIdle: true},
			StartedAt: started,
			EndedAt:   ended,
			Duration:  ended.Sub(started),
		},
		Reason: session.ReasonShutdown,
	}); err != nil {
		t.Fatalf("record idle transition: %v", err)
	}

	var appCount int
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT COUNT(*) FROM applications`).Scan(&appCount); err != nil {
		t.Fatalf("query applications: %v", err)
	}
	if appCount != 0 {
		t.Fatalf("expected no application for idle event, got %d", appCount)
	}

	var applicationID sql.NullInt64
	var reasonID int64
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT application_id, transition_reason_id FROM transition_events`).Scan(&applicationID, &reasonID); err != nil {
		t.Fatalf("query idle transition event: %v", err)
	}
	if applicationID.Valid {
		t.Fatalf("expected null application_id for idle event, got %d", applicationID.Int64)
	}
	if reasonID != 5 {
		t.Fatalf("expected shutdown reason id 5, got %d", reasonID)
	}

	var idle bool
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT idle FROM transition_event_metadata`).Scan(&idle); err != nil {
		t.Fatalf("query idle metadata: %v", err)
	}
	if !idle {
		t.Fatal("expected idle metadata to be true")
	}
}

func TestDatabaseRecordSkipsOpenAndStartTransitions(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	reporter := NewDatabaseReporter(database.WriteConn)

	if _, err := reporter.Record(ctx, session.Transition{
		To:     &session.Session{Key: session.AppKey{AppName: "VSCode"}, StartedAt: time.Now()},
		Reason: session.ReasonStart,
	}); err != nil {
		t.Fatalf("record start transition: %v", err)
	}
	if _, err := reporter.Record(ctx, session.Transition{
		From:   &session.Session{Key: session.AppKey{AppName: "VSCode"}, StartedAt: time.Now()},
		Reason: session.ReasonFocusChange,
	}); err != nil {
		t.Fatalf("record open transition: %v", err)
	}

	var eventCount int
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT COUNT(*) FROM transition_events`).Scan(&eventCount); err != nil {
		t.Fatalf("query transition events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected no transition events, got %d", eventCount)
	}
}

func TestDatabaseRecordIndexesPersistedTransitionEvent(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	indexer := &recordingTransitionEventIndexer{}
	reporter := NewDatabaseReporterWithIndexer(database.WriteConn, indexer)

	started := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(5 * time.Minute)

	recordedEventID, err := reporter.Record(ctx, session.Transition{
		From: &session.Session{
			Key:       session.AppKey{AppName: "VSCode"},
			StartedAt: started,
			EndedAt:   ended,
			Duration:  ended.Sub(started),
		},
		Reason: session.ReasonFocusChange,
	})
	if err != nil {
		t.Fatalf("record transition: %v", err)
	}

	var eventID int64
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT id FROM transition_events`).Scan(&eventID); err != nil {
		t.Fatalf("query transition event: %v", err)
	}
	if len(indexer.transitionEventIDs) != 1 || indexer.transitionEventIDs[0] != eventID {
		t.Fatalf("expected indexed event id %d, got %#v", eventID, indexer.transitionEventIDs)
	}
	if recordedEventID != eventID {
		t.Fatalf("expected returned event id %d, got %d", eventID, recordedEventID)
	}
}

func TestDatabaseRecordReturnsIndexingErrors(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	reporter := NewDatabaseReporterWithIndexer(database.WriteConn, &recordingTransitionEventIndexer{err: fmt.Errorf("embed failed")})

	started := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(5 * time.Minute)

	_, err := reporter.Record(ctx, session.Transition{
		From: &session.Session{
			Key:       session.AppKey{AppName: "VSCode"},
			StartedAt: started,
			EndedAt:   ended,
			Duration:  ended.Sub(started),
		},
		Reason: session.ReasonFocusChange,
	})
	if err == nil {
		t.Fatal("expected indexing error")
	}
	if !strings.Contains(err.Error(), "index transition event") || !strings.Contains(err.Error(), "embed failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	var eventCount int
	if err := database.WriteConn.QueryRowContext(ctx, `SELECT COUNT(*) FROM transition_events`).Scan(&eventCount); err != nil {
		t.Fatalf("query transition events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected persisted transition event despite indexing failure, got %d", eventCount)
	}
}

type recordingTransitionEventIndexer struct {
	transitionEventIDs []int64
	err                error
}

func (r *recordingTransitionEventIndexer) IndexTransitionEvent(_ context.Context, transitionEventID int64) (int64, error) {
	r.transitionEventIDs = append(r.transitionEventIDs, transitionEventID)
	if r.err != nil {
		return 0, r.err
	}
	return transitionEventID, nil
}

func newTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		Engine:         db.EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	return database
}

func assertSeededTransitionReasons(t *testing.T, ctx context.Context, conn *sql.DB) {
	t.Helper()

	expected := map[int64]string{
		1: "focus_change",
		2: "idle",
		3: "return_from_idle",
		4: "tab_change",
		5: "shutdown",
		6: "start",
	}

	rows, err := conn.QueryContext(ctx, `SELECT id, reason FROM transition_event_reasons ORDER BY id`)
	if err != nil {
		t.Fatalf("query seeded transition reasons: %v", err)
	}
	defer rows.Close()

	actual := map[int64]string{}
	for rows.Next() {
		var id int64
		var reason string
		if err := rows.Scan(&id, &reason); err != nil {
			t.Fatalf("scan seeded transition reason: %v", err)
		}
		actual[id] = reason
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate seeded transition reasons: %v", err)
	}

	if len(actual) != len(expected) {
		t.Fatalf("expected %d seeded transition reasons, got %d", len(expected), len(actual))
	}
	for id, reason := range expected {
		if actual[id] != reason {
			t.Fatalf("expected transition reason %d to be %q, got %q", id, reason, actual[id])
		}
	}
}
