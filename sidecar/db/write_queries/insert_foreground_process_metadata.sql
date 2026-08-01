-- name: InsertForegroundProcessMetadata :one
INSERT INTO foreground_process_metadata (foreground_process_id, browser, browser_vendor, idle, tab, cdp_url, latitude, longitude, public_ip, title_source, window_title)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;
