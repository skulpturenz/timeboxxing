-- name: GetOpenTimelineEvent :one
SELECT
  timeline.id AS transition_event_id,
  applications.name AS application_name,
  applications.identifier AS application_identifier,
  applications.path AS application_path,
  fp0.created_at_utc AS started_at,
  foreground_process_metadata.browser,
  foreground_process_metadata.tab,
  foreground_process_metadata.idle,
  foreground_process_metadata.cdp_url,
  fp0.application_id,
  fp0.pid
FROM timeline
JOIN foreground_processes fp0 ON fp0.id = timeline.initial_foreground_process_id
LEFT JOIN applications ON applications.id = fp0.application_id
LEFT JOIN foreground_process_metadata ON foreground_process_metadata.foreground_process_id = fp0.id
WHERE timeline.end_foreground_process_id IS NULL
ORDER BY timeline.id DESC
LIMIT 1;
