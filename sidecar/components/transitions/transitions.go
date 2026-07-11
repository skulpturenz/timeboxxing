package transitions

import (
	"context"
	"sync"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// writeTxRunner runs a function inside a serialized write transaction. *db.Database satisfies it.
type writeTxRunner interface {
	WriteTx(ctx context.Context, fn func(*writequeries.Queries) error) error
}

type Service struct {
	readQuerier readqueries.Querier
	writeTx     writeTxRunner

	mu               sync.Mutex
	nextSubscriberID int64
	subscribers      map[int64]subscriber
}

type serviceKey struct{}

func RegisterService(registry *services.Services[any, any], service *Service) {
	services.Set(registry, serviceKey{}, service)
}

func ServiceFromServices(registry *services.Services[any, any]) (*Service, bool) {
	service, ok := services.Get[*Service](registry, serviceKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func NewService(registry *services.Services[any, any]) *Service {
	database, _ := db.FromServices(registry)
	service := &Service{
		subscribers: map[int64]subscriber{},
	}
	if database != nil {
		service.readQuerier = database.ReadQuerier
		service.writeTx = database.WriteQuerier
	}
	RegisterService(registry, service)
	return service
}
