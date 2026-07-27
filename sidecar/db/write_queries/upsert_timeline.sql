-- name: UpsertTimeline :one
INSERT INTO timeline(initial_foreground_process_id, end_foreground_process_id)
VALUES (
    COALESCE(CAST(sqlc.narg(initial_foreground_process_id) AS INTEGER), (SELECT id FROM foreground_processes ORDER BY created_at_utc DESC LIMIT 1)),
    sqlc.narg(end_foreground_process_id)
)
ON CONFLICT(initial_foreground_process_id) DO UPDATE SET
    end_foreground_process_id = excluded.end_foreground_process_id
RETURNING id;
