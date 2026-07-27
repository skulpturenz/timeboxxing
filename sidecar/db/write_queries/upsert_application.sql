-- name: UpsertApplication :one
INSERT INTO applications (name, identifier, operating_system_id, path)
VALUES (?, ?, ?, ?)
ON CONFLICT(identifier, operating_system_id) DO UPDATE SET
  name = excluded.name,
  path = COALESCE(excluded.path, applications.path)
RETURNING id;
