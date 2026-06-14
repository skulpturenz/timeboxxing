package transitions

import (
	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
)

type Server struct {
	transitionsv1.UnimplementedTransitionsServiceServer

	transitions *componentTransitions.Service
}

type NewServerParams struct {
	Transitions *componentTransitions.Service
}

func NewServer(params NewServerParams) *Server {
	return &Server{transitions: params.Transitions}
}
