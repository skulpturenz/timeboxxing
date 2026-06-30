package projects

import (
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
)

func projectToProto(row queries.Project) *projectsv1.Project {
	return &projectsv1.Project{
		Id:              row.ID,
		Name:            row.Name,
		ColorArgb:       row.ColorArgb,
		Client:          row.Client,
		HourlyRateCents: row.HourlyRateCents,
	}
}

func projectsToProto(rows []queries.Project) []*projectsv1.Project {
	out := make([]*projectsv1.Project, 0, len(rows))
	for _, row := range rows {
		out = append(out, projectToProto(row))
	}
	return out
}

func trimmedProjectName(name string) string {
	return strings.TrimSpace(name)
}
