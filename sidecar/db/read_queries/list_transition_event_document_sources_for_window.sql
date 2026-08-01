-- name: ListTransitionEventDocumentSourcesForWindow :many
SELECT
  timeline.id AS transition_event_id,
  applications.name AS application_name,
  fp0.created_at_utc AS started_at,
  fp1.created_at_utc AS ended_at,
  foreground_process_metadata.browser,
  foreground_process_metadata.tab,
  foreground_process_metadata.idle,
  foreground_process_metadata.cdp_url
FROM timeline
JOIN foreground_processes fp0 ON fp0.id = timeline.initial_foreground_process_id
JOIN foreground_processes fp1 ON fp1.id = timeline.end_foreground_process_id
LEFT JOIN applications ON applications.id = fp0.application_id
LEFT JOIN foreground_process_metadata ON foreground_process_metadata.foreground_process_id = fp0.id
-- created_at_utc is stored with whatever offset the observation carried while the bounds arrive
-- normalised to UTC, so both the comparisons and the ordering go through unixepoch(): as text the
-- two are wall clocks rather than instants. ids are handed out in observation order, so ordering by
-- timeline.id is the same sequence and does not need the conversion
--
-- the leading CAST(... AS TIMESTAMP) tests are only a type anchor for sqlc, which otherwise infers
-- the bounds as interface{} once no comparison against the column types them; both bounds are
-- required, so the tests are always true
WHERE CAST(sqlc.arg('window_ended_at') AS TIMESTAMP) IS NOT NULL
  AND CAST(sqlc.arg('window_started_at') AS TIMESTAMP) IS NOT NULL
  AND unixepoch(fp0.created_at_utc, 'subsec') < unixepoch(sqlc.arg('window_ended_at'), 'subsec')
  AND unixepoch(fp1.created_at_utc, 'subsec') > unixepoch(sqlc.arg('window_started_at'), 'subsec')
ORDER BY timeline.id ASC;
