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
