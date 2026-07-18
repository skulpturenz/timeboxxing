package timeline

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func TestProjectReconcilesTimeline(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service := newProjector(t, database, 5*time.Second)
	base := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)

	// First observation opens the current entry.
	mustProject(t, ctx, service, appSample("A", "com.a", 1, base))
	assertTimelineCounts(t, ctx, database, 1, 1)

	// Same app within the granularity keeps extending the open entry (no new row).
	mustProject(t, ctx, service, appSample("A", "com.a", 1, base.Add(3*time.Second)))
	assertTimelineCounts(t, ctx, database, 1, 1)

	// A different app after the granularity closes the previous entry and opens a new one, sharing the
	// boundary foreground_processes row.
	mustProject(t, ctx, service, appSample("B", "com.b", 2, base.Add(10*time.Second)))
	assertTimelineCounts(t, ctx, database, 2, 1)
	assertSharedBoundary(t, ctx, database)

	// A different app within the granularity treats the previous (open) entry as a flicker: it is
	// deleted, so the total stays at two (A closed + the new open entry).
	mustProject(t, ctx, service, appSample("C", "com.c", 3, base.Add(12*time.Second)))
	assertTimelineCounts(t, ctx, database, 2, 1)
	assertOpenApp(t, ctx, database, "C")

	// Going idle is a distinct key and opens a fresh entry after the granularity.
	mustProject(t, ctx, service, idleSample(base.Add(30*time.Second)))
	assertTimelineCounts(t, ctx, database, 3, 1)
}

func newProjector(t *testing.T, database *db.Database, granularity time.Duration) *Service {
	t.Helper()
	registry := services.New()
	db.Register(registry, database)
	componentTransitions.NewService(registry)
	return NewService(registry, Options{Granularity: granularity})
}

func newTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	database, err := db.New(ctx, db.Options{
		DSN: db.NewDSN(filepath.Join(t.TempDir(), "timeline.db")),
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

func appSample(name, identifier string, pid int32, at time.Time) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{
		AppName:       &name,
		AppIdentifier: &identifier,
		PID:           &pid,
		Timestamp:     at,
	}
}

func idleSample(at time.Time) monitor.ForegroundProcess {
	return monitor.ForegroundProcess{Idle: true, Timestamp: at}
}

func mustProject(t *testing.T, ctx context.Context, service *Service, fp monitor.ForegroundProcess) {
	t.Helper()
	if err := service.Project(ctx, fp); err != nil {
		t.Fatalf("project sample: %v", err)
	}
}

func assertTimelineCounts(t *testing.T, ctx context.Context, database *db.Database, wantTotal, wantOpen int) {
	t.Helper()
	total := scanInt(t, ctx, database.ReadConn, "SELECT COUNT(*) FROM timeline")
	open := scanInt(t, ctx, database.ReadConn, "SELECT COUNT(*) FROM timeline WHERE end_foreground_process_id IS NULL")
	if total != wantTotal || open != wantOpen {
		t.Fatalf("timeline counts = (total %d, open %d), want (total %d, open %d)", total, open, wantTotal, wantOpen)
	}
}

// assertSharedBoundary checks that the closed entry's end boundary is the open entry's initial
// boundary — consecutive entries share a single foreground_processes row (no gap).
func assertSharedBoundary(t *testing.T, ctx context.Context, database *db.Database) {
	t.Helper()
	var closedEnd, openInitial int64
	row := database.ReadConn.QueryRowContext(ctx, `
		SELECT
		  (SELECT end_foreground_process_id FROM timeline WHERE end_foreground_process_id IS NOT NULL ORDER BY id DESC LIMIT 1),
		  (SELECT initial_foreground_process_id FROM timeline WHERE end_foreground_process_id IS NULL ORDER BY id DESC LIMIT 1)`)
	if err := row.Scan(&closedEnd, &openInitial); err != nil {
		t.Fatalf("scan boundary ids: %v", err)
	}
	if closedEnd != openInitial {
		t.Fatalf("expected shared boundary, closed end %d != open initial %d", closedEnd, openInitial)
	}
}

func assertOpenApp(t *testing.T, ctx context.Context, database *db.Database, wantName string) {
	t.Helper()
	var name sql.NullString
	row := database.ReadConn.QueryRowContext(ctx, `
		SELECT applications.name
		FROM timeline
		JOIN foreground_processes fp ON fp.id = timeline.initial_foreground_process_id
		LEFT JOIN applications ON applications.id = fp.application_id
		WHERE timeline.end_foreground_process_id IS NULL
		ORDER BY timeline.id DESC LIMIT 1`)
	if err := row.Scan(&name); err != nil {
		t.Fatalf("scan open app: %v", err)
	}
	if !name.Valid || name.String != wantName {
		t.Fatalf("open app = %q, want %q", name.String, wantName)
	}
}

func scanInt(t *testing.T, ctx context.Context, conn *sql.DB, query string) int {
	t.Helper()
	var value int
	if err := conn.QueryRowContext(ctx, query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}
