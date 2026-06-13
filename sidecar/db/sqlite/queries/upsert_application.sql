-- name: UpsertApplication :one
INSERT INTO applications (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING id;
