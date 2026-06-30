package timesheets

import (
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	timesheetsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/timesheets/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	timesheetsv1.UnimplementedTimesheetsServiceServer

	database *db.Database
	querier  queries.Querier
}

type serverKey struct{}

func RegisterServer(registry *services.Services[any, any], server *Server) {
	services.Set(registry, serverKey{}, server)
}

func ServerFromServices(registry *services.Services[any, any]) (*Server, bool) {
	service, ok := services.Get[*Server](registry, serverKey{})
	if !ok {
		return nil, false
	}
	return service.Unwrap(), true
}

func NewServer(registry *services.Services[any, any]) *Server {
	database, _ := db.FromServices(registry)
	var querier queries.Querier
	if database != nil {
		querier = database.WriteQuerier
	}
	server := &Server{
		database: database,
		querier:  querier,
	}
	RegisterServer(registry, server)
	return server
}
