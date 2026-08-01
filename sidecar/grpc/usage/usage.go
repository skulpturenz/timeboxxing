package usage

import (
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

// The timeline queries resolve their dependencies from the registry per call, so the server carries
// the registry rather than a component handle.
type Server struct {
	usagev1.UnimplementedUsageServiceServer

	registry *services.Services[any, any]
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
	server := &Server{registry: registry}
	RegisterServer(registry, server)
	return server
}
