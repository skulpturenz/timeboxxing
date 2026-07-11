-- name: CreateForegroundProcessMetadata :exec
INSERT INTO foreground_process_metadata (foreground_process_id, browser, idle, tab, cdp_url)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(foreground_process_id) DO NOTHING;
