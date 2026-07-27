package projects

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxProjectColorArgb = int64(0xFFFFFFFF)
	// costingTypeHourlyID matches the project_costing_types seed (db/seeds/project_costing_types).
	costingTypeHourlyID = 1
)

func (s *Server) CreateProject(ctx context.Context, req *projectsv1.CreateProjectRequest) (*projectsv1.Project, error) {
	if s.writeTx == nil {
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

	// Resolve the palette colour to its project_colors id. Colours outside the seeded palette leave
	// project_colors_id NULL (the colour is simply not persisted).
	var colorID *int64
	if s.readQuerier != nil {
		id, err := s.readQuerier.GetProjectColorIDByColor(ctx, colorARGB)
		switch {
		case err == nil:
			colorID = &id
		case errors.Is(err, sql.ErrNoRows):
			// leave NULL
		default:
			return nil, status.Errorf(codes.Internal, "resolve project colour: %v", err)
		}
	}

	// The id is assigned by SQLite (rowid). A duplicate name trips the name UNIQUE constraint.
	var projectID int64
	if err := s.writeTx.WriteTx(ctx, func(q *writequeries.Queries) error {
		id, err := q.CreateProject(ctx, name)
		if err != nil {
			return err
		}
		projectID = id
		costingTypeID := int64(costingTypeHourlyID)
		return q.CreateProjectDetails(ctx, writequeries.CreateProjectDetailsParams{
			ProjectsID:      &id,
			ProjectColorsID: colorID,
			CostingTypeID:   &costingTypeID,
			Rate:            nil,
		})
	}); err != nil {
		if isUniqueConstraintErr(err) {
			return nil, status.Error(codes.AlreadyExists, "A project with this name already exists.")
		}
		return nil, status.Errorf(codes.Internal, "create project: %v", err)
	}

	return &projectsv1.Project{
		Id:              projectID,
		Name:            name,
		ColorArgb:       colorARGB,
		Client:          "",
		HourlyRateCents: 0,
	}, nil
}

func isUniqueConstraintErr(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}
