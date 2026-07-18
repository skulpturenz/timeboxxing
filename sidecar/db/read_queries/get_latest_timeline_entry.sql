-- name: GetLatestTimelineEntry :one
SELECT
  timeline.id AS timeline_id,
  timeline.initial_foreground_process_id,
  timeline.end_foreground_process_id,
  fp0.pid AS initial_pid,
  fp0.created_at_utc AS initial_created_at,
  foreground_process_metadata.tab AS initial_tab,
  foreground_process_metadata.idle AS initial_idle
FROM timeline
JOIN foreground_processes fp0 ON fp0.id = timeline.initial_foreground_process_id
LEFT JOIN foreground_process_metadata ON foreground_process_metadata.foreground_process_id = fp0.id
ORDER BY timeline.id DESC
LIMIT 1;
