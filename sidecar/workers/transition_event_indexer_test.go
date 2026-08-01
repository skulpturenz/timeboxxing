package workers

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goptics/sqliteq"
	componentTimeline "github.com/skulpturenz/timeboxxing/sidecar/components/timeline"
	timelineModels "github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
	"github.com/skulpturenz/timeboxxing/sidecar/logging"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type workerFakeEmbedder struct{}

func (workerFakeEmbedder) Model() string { return "worker-fake-embedding" }

func (workerFakeEmbedder) Dimension() int { return semantic.StoreEmbeddingDimension }

func (workerFakeEmbedder) Embed(_ context.Context, input string) ([]float32, error) {
	values := make([]float32, semantic.StoreEmbeddingDimension)
	if strings.Contains(input, "Google Chrome") {
		values[0] = 1
	} else {
		values[1] = 1
	}
	return values, nil
}

func TestTransitionEventBackfillQueueIndexesEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsn := db.NewDSN(filepath.Join(t.TempDir(), "backfill-workers.db"))
	dsn.SetJournalMode(enumsjournalmode.WAL)
	dsn.EnableFK()
	dsn.SetBusyTimeout(5 * time.Second)
	database, err := db.New(ctx, db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	queueDSN := dsn.String()
	manager := sqliteq.New(queueDSN)
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("close queue manager: %v", err)
		}
	})

	inChan := make(chan TransitionEventReported)
	outChan, err := queue.New[TransitionEventReported](ctx, queue.QueueOptions[TransitionEventReported]{
		Manager: manager,
		Name:    TransitionEventReportedQueueName.String(),
		InChan:  inChan,
	})
	if err != nil {
		t.Fatalf("create transition event reported queue: %v", err)
	}

	registry := services.New()
	logging.RegisterLogger(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))
	db.Register(registry, database)
	RegisterQueues(registry, Queues{
		TransitionEventReportedIn:  inChan,
		TransitionEventReportedOut: outChan,
	})
	semantic.RegisterRuntime(registry, &semantic.Runtime{
		Indexer: semantic.NewIndexer(database.WriteQuerier, database.ReadQuerier, workerFakeEmbedder{}, 1),
	})

	runtime := NewRuntime(registry)
	cleanupIndexer := runtime.TransitionEventIndexerWorker(ctx, outChan)
	// cancel before draining: the indexer stops on ctx cancellation, so cleanup can complete.
	defer func() {
		cancel()
		cleanupIndexer()
	}()

	// the ingest chains observations: each one closes the entry the previous one opened, so two
	// observations are the minimum that produces a closed — and therefore indexable — entry
	seedIndexerObservations(t, ctx, registry,
		browserForegroundProcess("Google Chrome", "Backfill Notes", "https://example.com/backfill", time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC)),
		browserForegroundProcess("Google Chrome", "Follow up", "https://example.com/follow-up", time.Date(2026, 6, 13, 11, 5, 0, 0, time.UTC)),
	)
	eventID := latestClosedTimelineID(t, ctx, database.ReadConn)

	enqueuer := NewTransitionEventReportedEnqueuer(inChan)
	if err := enqueuer.EnqueueTransitionEvent(ctx, eventID); err != nil {
		t.Fatalf("enqueue backfill transition event: %v", err)
	}

	waitForWorkerRowCount(t, ctx, database.ReadConn, "timeline_semantic_documents", 7)
	waitForWorkerRowCount(t, ctx, database.ReadConn, "timeline_embeddings", 7)
}

func countWorkerRows(t *testing.T, ctx context.Context, conn *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func waitForWorkerRowCount(t *testing.T, ctx context.Context, conn *sql.DB, table string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := countWorkerRows(t, ctx, conn, table); got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %s row count %d, got %d", table, want, countWorkerRows(t, ctx, conn, table))
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// seedIndexerObservations writes observations through the real ingest command so the timeline
// entries under test are shaped exactly as production writes them.
func seedIndexerObservations(t *testing.T, ctx context.Context, registry *services.Services[any, any], observations ...timelineModels.ForegroundProcess) {
	t.Helper()

	var previous *timelineModels.ForegroundProcess
	for i := range observations {
		cmd := componentTimeline.CommandUpsertForegroundProcess{
			PreviousProcess: previous,
			ActiveProcess:   observations[i],
		}
		if err := cmd.Exec(ctx, registry); err != nil {
			t.Fatalf("seed observation %d: %v", i, err)
		}

		previous = &observations[i]
	}
}

func browserForegroundProcess(name string, tab string, cdpURL string, timestamp time.Time) timelineModels.ForegroundProcess {
	identifier := "com." + strings.ToLower(strings.ReplaceAll(name, " ", "."))
	path := "/Applications/" + name + ".app"
	pid := int64(4242)

	return timelineModels.ForegroundProcess{
		AppName:       &name,
		AppIdentifier: &identifier,
		AppPath:       &path,
		PID:           &pid,
		Timestamp:     timestamp,
		Enrichments: timelineModels.Enrichments{
			Browser: timelineModels.Browser{Vendor: name, Tab: tab, CdpURL: cdpURL},
		},
	}
}

// latestClosedTimelineID is the newest entry that has both endpoints — an open entry has no final
// observation, and the semantic document source requires one.
func latestClosedTimelineID(t *testing.T, ctx context.Context, conn *sql.DB) int64 {
	t.Helper()

	var id int64
	query := "SELECT id FROM timeline WHERE end_foreground_process_id IS NOT NULL ORDER BY id DESC LIMIT 1"
	if err := conn.QueryRowContext(ctx, query).Scan(&id); err != nil {
		t.Fatalf("latest closed timeline entry: %v", err)
	}

	return id
}
