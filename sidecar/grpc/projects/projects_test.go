package projects

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/db"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateProjectTrimsStoresDefaultsAndLists(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestProjectsServer(t, ctx)
	defer cleanup()

	project, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{
		Name:      "  Client Work  ",
		ColorArgb: 0xFF00FFEE,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.GetId() != "client-work" {
		t.Fatalf("expected slug id client-work, got %q", project.GetId())
	}
	if project.GetName() != "Client Work" {
		t.Fatalf("expected trimmed name, got %q", project.GetName())
	}
	if project.GetClient() != "" {
		t.Fatalf("expected empty client, got %q", project.GetClient())
	}
	if project.GetHourlyRateCents() != 0 {
		t.Fatalf("expected zero hourly rate, got %d", project.GetHourlyRateCents())
	}

	listed, err := server.ListProjects(ctx, &projectsv1.ListProjectsRequest{})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(listed.GetProjects()) != 1 {
		t.Fatalf("expected 1 project, got %d", len(listed.GetProjects()))
	}
	if listed.GetProjects()[0].GetId() != project.GetId() {
		t.Fatalf("expected listed project id %q, got %q", project.GetId(), listed.GetProjects()[0].GetId())
	}
}

func TestCreateProjectGeneratesUniqueSlugID(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestProjectsServer(t, ctx)
	defer cleanup()

	first, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "Client", ColorArgb: 1})
	if err != nil {
		t.Fatalf("create first project: %v", err)
	}
	second, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "Client!", ColorArgb: 2})
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}

	if first.GetId() != "client" {
		t.Fatalf("expected first id client, got %q", first.GetId())
	}
	if second.GetId() != "client-2" {
		t.Fatalf("expected second id client-2, got %q", second.GetId())
	}
}

func TestCreateProjectRejectsBlankAndDuplicateNames(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestProjectsServer(t, ctx)
	defer cleanup()

	if _, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "  ", ColorArgb: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for blank name, got %v", err)
	}
	if _, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "Alpha", ColorArgb: 1}); err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	if _, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "alpha", ColorArgb: 2}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected already exists for duplicate name, got %v", err)
	}
}

func TestDeleteProjectRemovesProjectAndUnknownIDIsSuccess(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestProjectsServer(t, ctx)
	defer cleanup()

	project, err := server.CreateProject(ctx, &projectsv1.CreateProjectRequest{Name: "Client", ColorArgb: 1})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := server.DeleteProject(ctx, &projectsv1.DeleteProjectRequest{Id: project.GetId()}); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if _, err := server.DeleteProject(ctx, &projectsv1.DeleteProjectRequest{Id: project.GetId()}); err != nil {
		t.Fatalf("delete project again: %v", err)
	}

	listed, err := server.ListProjects(ctx, &projectsv1.ListProjectsRequest{})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(listed.GetProjects()) != 0 {
		t.Fatalf("expected no projects, got %d", len(listed.GetProjects()))
	}
}

func newTestProjectsServer(t *testing.T, ctx context.Context) (*Server, func()) {
	t.Helper()

	database, err := db.New(ctx, db.Options{
		Engine: db.EngineSqlite,
		DSN:    filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	registry := services.New()
	db.Register(registry, database)
	server := NewServer(registry)

	return server, func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	}
}
