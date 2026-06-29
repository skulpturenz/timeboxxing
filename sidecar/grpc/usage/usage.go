package usage

import (
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	usagev1.UnimplementedUsageServiceServer

	usage *componentUsage.Service
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
	usage, _ := componentUsage.ServiceFromServices(registry)
	server := &Server{usage: usage}
	RegisterServer(registry, server)
	return server
}
