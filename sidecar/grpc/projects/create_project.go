package projects

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/mattn/go-sqlite3"
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxProjectColorArgb = int64(0xFFFFFFFF)

func (s *Server) CreateProject(ctx context.Context, req *projectsv1.CreateProjectRequest) (*projectsv1.Project, error) {
	if s.writeQuerier == nil {
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

	// Insert with a slug id, retrying with a numeric suffix on id collisions. CreateProject is
	// `ON CONFLICT(id) DO NOTHING RETURNING`, so an id clash returns no row (sql.ErrNoRows). A
	// duplicate name is not in the conflict target, so it surfaces as a UNIQUE constraint error.
	now := time.Now().UTC()
	base := projectSlug(name)
	candidate := base
	for suffix := 2; ; suffix++ {
		project, err := s.writeQuerier.CreateProject(ctx, writequeries.CreateProjectParams{
			ID:        candidate,
			Name:      name,
			ColorArgb: colorARGB,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err == nil {
			return projectToProto(writeProjectToRead(project)), nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			candidate = base + "-" + strconv.Itoa(suffix)
			continue
		}
		if isUniqueConstraintErr(err) {
			return nil, status.Error(codes.AlreadyExists, "A project with this name already exists.")
		}
		return nil, status.Errorf(codes.Internal, "create project: %v", err)
	}
}

func isUniqueConstraintErr(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

// writeProjectToRead converts a write-package project row into the read-package struct the proto
// mappers are typed on (identical fields; the two sqlc packages generate distinct types).
func writeProjectToRead(p writequeries.Project) readqueries.Project {
	return readqueries.Project{
		ID:              p.ID,
		Name:            p.Name,
		ColorArgb:       p.ColorArgb,
		Client:          p.Client,
		HourlyRateCents: p.HourlyRateCents,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
