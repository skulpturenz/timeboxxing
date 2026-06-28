package usage

import (
	componentUsage "github.com/skulpturenz/timeboxxing/sidecar/components/usage"
	usagev1 "github.com/skulpturenz/timeboxxing/sidecar/gen/usage/v1"
)

type Server struct {
	usagev1.UnimplementedUsageServiceServer

	usage *componentUsage.Service
}

type NewServerParams struct {
	Usage *componentUsage.Service
}

func NewServer(params NewServerParams) *Server {
	return &Server{usage: params.Usage}
}
