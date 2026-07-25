-- name: UpsertForegroundProcess :one
INSERT INTO foreground_processes(application_id, pid, created_at_utc)
VALUES (?, ?, ?)
ON CONFLICT(created_at_utc) DO UPDATE SET
    application_id = excluded.application_id,
    pid = excluded.pid
RETURNING id;
