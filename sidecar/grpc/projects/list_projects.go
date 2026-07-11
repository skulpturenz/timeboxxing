package projects

import (
	"context"

	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListProjects(ctx context.Context, req *projectsv1.ListProjectsRequest) (*projectsv1.ListProjectsResponse, error) {
	if s.readQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is unavailable")
	}

	rows, err := s.readQuerier.ListProjects(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list projects: %v", err)
	}
	return &projectsv1.ListProjectsResponse{Projects: projectsToProto(rows)}, nil
}
