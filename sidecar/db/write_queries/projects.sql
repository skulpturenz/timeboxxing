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
ON CONFLICT (id) DO NOTHING
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
