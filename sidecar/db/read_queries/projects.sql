-- name: ListProjects :many
SELECT
  projects.id,
  projects.name,
  project_colors.color AS color_argb,
  COALESCE(project_details.rate, 0) AS hourly_rate_cents
FROM projects
LEFT JOIN project_details ON project_details.projects_id = projects.id
LEFT JOIN project_colors ON project_colors.id = project_details.project_colors_id
ORDER BY projects.name COLLATE NOCASE, projects.id;

-- name: GetProjectColorIDByColor :one
SELECT id
FROM project_colors
WHERE color = ?;
