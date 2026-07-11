-- name: GetSemanticEventDocumentSource :one
SELECT
  timeline.id AS transition_event_id,
  applications.name AS application_name,
  fp0.application_id AS application_id,
  fp0.created_at_utc AS started_at,
  fp1.created_at_utc AS ended_at,
  foreground_process_metadata.browser,
  foreground_process_metadata.tab,
  foreground_process_metadata.idle,
  foreground_process_metadata.cdp_url,
  (
    SELECT pfp.application_id
    FROM timeline pt
    JOIN foreground_processes pfp ON pfp.id = pt.initial_foreground_process_id
    WHERE pt.id < timeline.id
    ORDER BY pt.id DESC
    LIMIT 1
  ) AS prev_application_id,
  (
    SELECT pmeta.idle
    FROM timeline pt
    LEFT JOIN foreground_process_metadata pmeta ON pmeta.foreground_process_id = pt.initial_foreground_process_id
    WHERE pt.id < timeline.id
    ORDER BY pt.id DESC
    LIMIT 1
  ) AS prev_idle
FROM timeline
JOIN foreground_processes fp0 ON fp0.id = timeline.initial_foreground_process_id
JOIN foreground_processes fp1 ON fp1.id = timeline.end_foreground_process_id
LEFT JOIN applications ON applications.id = fp0.application_id
LEFT JOIN foreground_process_metadata ON foreground_process_metadata.foreground_process_id = fp0.id
WHERE timeline.id = ?;
