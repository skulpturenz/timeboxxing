package workers

import (
	"database/sql"
	"log/slog"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/queue"
)

type WorkerServices struct {
	ReadQueries            queries.Querier
	WriteQueries           queries.Querier
	ReadConn               *sql.DB
	WriteConn              *sql.DB
	Logger                 *slog.Logger
	Queues                 WorkerQueues
	TransitionEventIndexer reporter.TransitionEventIndexer
	Transitions            *componentTransitions.Service
}

type WorkerQueues struct {
	TransitionEventQueue         *queue.Queue[reporter.TransitionEvent]
	TransitionEventReportedQueue *queue.Queue[TransitionEventReported]
	TransitionEventIndexedQueue  *queue.Queue[TransitionEventIndexed]
}
