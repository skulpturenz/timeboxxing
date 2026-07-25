-- name: InsertForegroundProcessMetadata :one
INSERT INTO foreground_process_metadata (foreground_process_id, browser, idle, tab, cdp_url, latitude, longitude, public_ip)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;
