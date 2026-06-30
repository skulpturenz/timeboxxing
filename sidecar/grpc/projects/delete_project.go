package projects

import (
	"context"
	"strings"

	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) DeleteProject(ctx context.Context, req *projectsv1.DeleteProjectRequest) (*projectsv1.DeleteProjectResponse, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is unavailable")
	}

	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "project id is required")
	}
	if err := s.querier.DeleteProject(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete project: %v", err)
	}
	return &projectsv1.DeleteProjectResponse{}, nil
}
