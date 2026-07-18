package timeline

import (
	"context"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// defaultGranularity is the flicker threshold: a timeline entry younger than this when it is
// superseded is discarded rather than recorded (see Project).
const defaultGranularity = 5 * time.Second

// writeTxRunner runs a function inside a serialized write transaction. *db.Database's WriteQuerier
// satisfies it.
type writeTxRunner interface {
	WriteTx(ctx context.Context, fn func(*writequeries.Queries) error) error
}

// transitionPublisher fans a finalized timeline entry out to live usage subscribers.
// *transitions.Service satisfies it.
type transitionPublisher interface {
	PublishTransitionEvent(ctx context.Context, params componentTransitions.PublishTransitionEventParams) error
}

// reportedEnqueuer enqueues a finalized timeline entry for semantic indexing.
// *workers.TransitionEventReportedEnqueuer satisfies it; it may be nil when indexing is disabled.
type reportedEnqueuer interface {
	EnqueueTransitionEvent(ctx context.Context, transitionEventID int64) error
}

type Options struct {
	// Granularity y: entries superseded before this age are treated as flicker and dropped.
	Granularity time.Duration
	// Clock is used only to stamp samples that arrive without a timestamp; defaults to time.Now.
	Clock func() time.Time
	// Enqueuer feeds finalized entries to the semantic indexer; nil disables enqueueing.
	Enqueuer reportedEnqueuer
}

// Service maintains a running projection of the foreground-process event store into the timeline:
// each observation is persisted, then reconciled against the latest timeline entry (extend the open
// entry, close it and open a new one, or delete a sub-granularity flicker).
type Service struct {
	readQuerier readqueries.Querier
	writeTx     writeTxRunner
	publisher   transitionPublisher
	enqueuer    reportedEnqueuer
	granularity time.Duration
	clock       func() time.Time
}

func NewService(registry *services.Services[any, any], opts Options) *Service {
	granularity := opts.Granularity
	if granularity <= 0 {
		granularity = defaultGranularity
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	service := &Service{
		enqueuer:    opts.Enqueuer,
		granularity: granularity,
		clock:       clock,
	}
	if database, ok := db.FromServices(registry); ok && database != nil {
		service.readQuerier = database.ReadQuerier
		service.writeTx = database.WriteQuerier
	}
	if publisher, ok := componentTransitions.ServiceFromServices(registry); ok {
		service.publisher = publisher
	}
	return service
}
