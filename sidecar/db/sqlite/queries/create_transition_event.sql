-- name: CreateTransitionEvent :one
INSERT INTO transition_events (application_id, started_at, ended_at)
VALUES (?, ?, ?)
RETURNING id;
