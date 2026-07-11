package projects

import (
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	projectsv1.UnimplementedProjectsServiceServer

	readQuerier  readqueries.Querier
	writeQuerier writequeries.Querier
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
	server := &Server{}
	if database != nil {
		server.readQuerier = database.ReadQuerier
		server.writeQuerier = database.WriteQuerier
	}
	RegisterServer(registry, server)
	return server
}
