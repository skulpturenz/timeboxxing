-- name: CreateTransitionEventNow :one
INSERT INTO transition_events (application_id, transition_reason_id)
VALUES (?, (SELECT id FROM transition_event_reasons WHERE reason = ?))
RETURNING id;
