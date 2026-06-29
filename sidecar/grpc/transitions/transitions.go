package transitions

import (
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type Server struct {
	transitionsv1.UnimplementedTransitionsServiceServer

	transitions *componentTransitions.Service
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
	transitions, _ := componentTransitions.ServiceFromServices(registry)
	server := &Server{transitions: transitions}
	RegisterServer(registry, server)
	return server
}
