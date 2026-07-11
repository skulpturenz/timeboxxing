-- name: UpsertApplication :one
INSERT INTO applications (name, operating_system_id, path)
VALUES (sqlc.arg('name'), sqlc.narg('operating_system_id'), sqlc.narg('path'))
ON CONFLICT(name) DO UPDATE SET
  name = excluded.name,
  operating_system_id = COALESCE(excluded.operating_system_id, applications.operating_system_id),
  path = COALESCE(excluded.path, applications.path)
RETURNING id;
