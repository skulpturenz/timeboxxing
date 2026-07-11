package projects

import (
	"strings"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	projectsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/projects/v1"
)

func projectToProto(row readqueries.ListProjectsRow) *projectsv1.Project {
	project := &projectsv1.Project{
		Id:              row.ID,
		Name:            row.Name,
		HourlyRateCents: row.HourlyRateCents,
	}
	if row.ColorArgb.Valid {
		project.ColorArgb = row.ColorArgb.Int64
	}
	return project
}

func projectsToProto(rows []readqueries.ListProjectsRow) []*projectsv1.Project {
	out := make([]*projectsv1.Project, 0, len(rows))
	for _, row := range rows {
		out = append(out, projectToProto(row))
	}
	return out
}

func trimmedProjectName(name string) string {
	return strings.TrimSpace(name)
}
