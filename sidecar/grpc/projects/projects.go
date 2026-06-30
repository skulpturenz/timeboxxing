package projects

import (
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	projectsv1.UnimplementedProjectsServiceServer

	querier queries.Querier
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
	server := &Server{querier: querier}
	RegisterServer(registry, server)
	return server
}
