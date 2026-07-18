-- name: UpsertApplication :one
INSERT INTO applications (name, identifier, operating_system_id, path)
VALUES (sqlc.arg('name'), sqlc.narg('identifier'), sqlc.narg('operating_system_id'), sqlc.narg('path'))
ON CONFLICT(identifier, operating_system_id) DO UPDATE SET
  name = excluded.name,
  path = COALESCE(excluded.path, applications.path)
RETURNING id;
