-- name: UpsertTimeline :one
INSERT INTO timeline(initial_foreground_process_id, end_foreground_process_id)
VALUES (
    -- the fallback is the observation before the one just written. it orders by id rather than by
    -- created_at_utc because ids are handed out in observation order, while the timestamp is text
    -- carrying whatever offset the observation was recorded in and does not sort chronologically
    -- across offsets
    COALESCE(CAST(sqlc.narg(initial_foreground_process_id) AS INTEGER), (SELECT id FROM foreground_processes ORDER BY id DESC LIMIT 1 OFFSET 1)),
    sqlc.narg(end_foreground_process_id)
)
ON CONFLICT(initial_foreground_process_id) DO UPDATE SET
    end_foreground_process_id = excluded.end_foreground_process_id
RETURNING id;
