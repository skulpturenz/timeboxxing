-- name: GetMetadata :one
SELECT key, value
FROM app_metadata
WHERE key = ?;

-- name: ListMetadata :many
SELECT key, value
FROM app_metadata
ORDER BY key;

-- name: SetMetadata :exec
INSERT INTO app_metadata (key, value)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: DeleteMetadata :exec
DELETE FROM app_metadata
WHERE key = ?;
