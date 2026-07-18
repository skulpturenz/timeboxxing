-- name: UpdateTimelineEnd :exec
UPDATE timeline SET end_foreground_process_id = ? WHERE id = ?;
