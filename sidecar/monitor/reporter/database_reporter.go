package reporter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/browser"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
)

// DatabaseReporter reads closed sessions from transitions and persists them.
type DatabaseReporter struct {
	conn    *sql.DB
	indexer TransitionEventIndexer
}

type TransitionEventIndexer interface {
	IndexTransitionEvent(ctx context.Context, transitionEventID int64) (int64, error)
}

// NewDatabaseReporter creates a database reporter using conn.
func NewDatabaseReporter(conn *sql.DB) *DatabaseReporter {
	return &DatabaseReporter{conn: conn}
}

// NewDatabaseReporterWithIndexer creates a database reporter that also indexes persisted events.
func NewDatabaseReporterWithIndexer(conn *sql.DB, indexer TransitionEventIndexer) *DatabaseReporter {
	return &DatabaseReporter{conn: conn, indexer: indexer}
}

// Run drains transitions until the channel closes or ctx is cancelled.
func (d *DatabaseReporter) Run(ctx context.Context, transitions <-chan session.Transition) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t, ok := <-transitions:
			if !ok {
				return nil
			}
			if err := d.Record(ctx, t); err != nil {
				return err
			}
		}
	}
}

// Record persists the closed source session from a transition.
func (d *DatabaseReporter) Record(ctx context.Context, t session.Transition) error {
	if t.From == nil || t.From.IsOpen() {
		return nil
	}

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transition event transaction: %w", err)
	}
	defer tx.Rollback()

	q := queries.New(tx)
	key := t.From.Key
	appName := strings.TrimSpace(key.AppName)
	applicationID := sql.NullInt64{}

	if !key.IsIdle && appName != "" {
		id, err := q.UpsertApplication(ctx, appName)
		if err != nil {
			return fmt.Errorf("upsert application %q: %w", appName, err)
		}
		applicationID = sql.NullInt64{Int64: id, Valid: true}
	}

	eventID, err := q.CreateTransitionEvent(ctx, queries.CreateTransitionEventParams{
		ApplicationID: applicationID,
		Reason:        string(t.Reason),
		StartedAt:     t.From.StartedAt,
		EndedAt:       t.From.EndedAt,
	})
	if err != nil {
		return fmt.Errorf("create transition event: %w", err)
	}

	if err := q.CreateTransitionEventMetadata(ctx, queries.CreateTransitionEventMetadataParams{
		TransitionEventID: eventID,
		Browser:           browser.IsBrowser(appName) != browser.BrowserNone,
		Tab:               stringPtr(key.TabTitle),
		Idle:              key.IsIdle,
		CdpUrl:            stringPtr(key.CDPURL),
	}); err != nil {
		return fmt.Errorf("create transition event metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transition event transaction: %w", err)
	}
	if d.indexer != nil {
		if _, err := d.indexer.IndexTransitionEvent(ctx, eventID); err != nil {
			return fmt.Errorf("index transition event: %w", err)
		}
	}

	return nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
