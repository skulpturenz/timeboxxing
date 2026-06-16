package workers

import (
	"database/sql"
	"log/slog"

	"github.com/goptics/varmq"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
)

type WorkerServices struct {
	ReadQueries            queries.Querier
	WriteQueries           queries.Querier
	ReadConn               *sql.DB
	WriteConn              *sql.DB
	Logger                 *slog.Logger
	Queues                 WorkerQueues
	TransitionEventIndexer reporter.TransitionEventIndexer
}

type WorkerQueues struct {
	TransitionEventQueue         varmq.PersistentQueue[any]
	TransitionEventReportedQueue varmq.PersistentQueue[any]
	TransitionEventIndexedQueue  varmq.PersistentQueue[any]
}
