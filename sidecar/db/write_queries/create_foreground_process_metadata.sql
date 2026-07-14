-- name: CreateForegroundProcessMetadata :exec
INSERT INTO foreground_process_metadata (foreground_process_id, browser, idle, tab, cdp_url, latitude, longitude, public_ip)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(foreground_process_id) DO NOTHING;
