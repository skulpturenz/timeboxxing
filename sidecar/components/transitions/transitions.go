package transitions

import (
	"sync"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type Service struct {
	querier queries.Querier

	mu               sync.Mutex
	nextSubscriberID int64
	subscribers      map[int64]subscriber
}

type NewServiceParams struct {
	Querier queries.Querier
}

func NewService(params NewServiceParams) *Service {
	return &Service{
		querier:     params.Querier,
		subscribers: map[int64]subscriber{},
	}
}
