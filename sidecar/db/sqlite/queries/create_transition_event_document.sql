-- name: CreateTransitionEventDocument :one
INSERT INTO transition_event_documents (transition_event_id, content, embedding_model, embedding_dimension, embedded_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id;
