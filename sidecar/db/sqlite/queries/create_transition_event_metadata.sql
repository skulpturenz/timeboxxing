-- name: CreateTransitionEventMetadata :exec
INSERT INTO transition_event_metadata (transition_event_id, browser, tab, idle, cdp_url)
VALUES (?, ?, ?, ?, ?);
