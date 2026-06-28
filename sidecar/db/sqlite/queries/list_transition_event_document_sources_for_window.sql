-- name: ListTransitionEventDocumentSourcesForWindow :many
SELECT
  transition_events.id AS transition_event_id,
  applications.name AS application_name,
  transition_event_reasons.reason AS reason,
  transition_events.started_at,
  transition_events.ended_at,
  transition_event_metadata.browser,
  transition_event_metadata.tab,
  transition_event_metadata.idle,
  transition_event_metadata.cdp_url
FROM transition_events
JOIN transition_event_reasons ON transition_event_reasons.id = transition_events.transition_reason_id
LEFT JOIN applications ON applications.id = transition_events.application_id
LEFT JOIN transition_event_metadata ON transition_event_metadata.transition_event_id = transition_events.id
WHERE transition_events.started_at < sqlc.arg('window_ended_at')
  AND transition_events.ended_at > sqlc.arg('window_started_at')
ORDER BY transition_events.started_at ASC, transition_events.id ASC;
