-- name: ListProjects :many
SELECT
  id,
  name,
  color_argb,
  client,
  hourly_rate_cents,
  created_at,
  updated_at
FROM projects
ORDER BY created_at, name COLLATE NOCASE, id;

-- name: CountProjectsByID :one
SELECT COUNT(*) FROM projects WHERE id = ?;

-- name: CountProjectsByName :one
SELECT COUNT(*) FROM projects WHERE name = ? COLLATE NOCASE;

-- name: CreateProject :one
INSERT INTO projects (
  id,
  name,
  color_argb,
  client,
  hourly_rate_cents,
  created_at,
  updated_at
)
VALUES (?, ?, ?, '', 0, ?, ?)
RETURNING
  id,
  name,
  color_argb,
  client,
  hourly_rate_cents,
  created_at,
  updated_at;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
