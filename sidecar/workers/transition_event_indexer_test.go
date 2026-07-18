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

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
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
	transitionEventReportedQueue, err := queue.New[TransitionEventReported](ctx, queue.QueueOptions{
		ConnectionString: queueDSN,
		QueueName:        TransitionEventReportedQueueName.String(),
	})
	if err != nil {
		t.Fatalf("create transition event reported queue: %v", err)
	}
	t.Cleanup(func() {
		if err := transitionEventReportedQueue.Close(); err != nil {
			t.Errorf("close transition event reported queue: %v", err)
		}
	})

	registry := services.New()
	logging.RegisterLogger(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))
	db.Register(registry, database)
	RegisterQueues(registry, Queues{TransitionEventReportedQueue: transitionEventReportedQueue})
	transitions := componentTransitions.NewService(registry)
	semantic.RegisterRuntime(registry, &semantic.Runtime{
		Indexer: semantic.NewIndexer(database.WriteQuerier, database.ReadQuerier, workerFakeEmbedder{}, 1),
	})

	runtime := NewRuntime(registry)
	cleanupIndexer := runtime.TransitionEventIndexerWorker(ctx, transitionEventReportedQueue)
	defer cleanupIndexer()

	tab := "Backfill Notes"
	url := "https://example.com/backfill"
	eventID, err := transitions.RecordTransitionEvent(ctx, componentTransitions.RecordTransitionEventParams{
		ApplicationName: "Google Chrome",
		Reason:          "focus_change",
		StartedAt:       time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC),
		EndedAt:         time.Date(2026, 6, 13, 11, 5, 0, 0, time.UTC),
		Browser:         true,
		Tab:             &tab,
		CDPURL:          &url,
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}

	enqueuer := NewTransitionEventReportedEnqueuer(transitionEventReportedQueue)
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
