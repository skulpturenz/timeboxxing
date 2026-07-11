-- name: UpsertApplication :one
INSERT INTO applications (name, platform_identifier, path)
VALUES (sqlc.arg('name'), sqlc.narg('platform_identifier'), sqlc.narg('path'))
ON CONFLICT(name) DO UPDATE SET
  name = excluded.name,
  platform_identifier = COALESCE(excluded.platform_identifier, applications.platform_identifier),
  path = COALESCE(excluded.path, applications.path)
RETURNING id;
