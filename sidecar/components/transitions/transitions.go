package transitions

import (
	"database/sql"
	"sync"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Service struct {
	readQuerier queries.Querier
	writeConn   *sql.DB

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
		service.writeConn = database.WriteConn
	}
	RegisterService(registry, service)
	return service
}
