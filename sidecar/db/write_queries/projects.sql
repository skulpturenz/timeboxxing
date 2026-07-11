-- name: CreateProject :one
INSERT INTO projects (name)
VALUES (?)
RETURNING id;

-- name: CreateProjectDetails :exec
INSERT INTO project_details (projects_id, project_colors_id, costing_type_id, rate)
VALUES (?, ?, ?, ?);

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
