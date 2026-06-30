package projects

import (
	"context"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxProjectColorArgb = int64(0xFFFFFFFF)

func (s *Server) CreateProject(ctx context.Context, req *projectsv1.CreateProjectRequest) (*projectsv1.Project, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is unavailable")
	}

	name := trimmedProjectName(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "project name is required")
	}
	colorARGB := req.GetColorArgb()
	if colorARGB < 0 || colorARGB > maxProjectColorArgb {
		return nil, status.Error(codes.InvalidArgument, "project colour is invalid")
	}

	count, err := s.querier.CountProjectsByName(ctx, name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check project name: %v", err)
	}
	if count > 0 {
		return nil, status.Error(codes.AlreadyExists, "A project with this name already exists.")
	}

	id, err := uniqueProjectID(ctx, s.querier, name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate project id: %v", err)
	}
	now := time.Now().UTC()
	project, err := s.querier.CreateProject(ctx, queries.CreateProjectParams{
		ID:        id,
		Name:      name,
		ColorArgb: colorARGB,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create project: %v", err)
	}
	return projectToProto(project), nil
}
