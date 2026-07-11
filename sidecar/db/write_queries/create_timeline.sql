-- name: CreateTimeline :one
INSERT INTO timeline (initial_foreground_process_id, end_foreground_process_id)
VALUES (?, ?)
RETURNING id;
