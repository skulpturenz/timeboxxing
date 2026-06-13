-- name: UpsertTransitionEventDocument :one
INSERT INTO transition_event_documents (transition_event_id, content, embedding_model, embedding_dimension, embedded_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(transition_event_id) DO UPDATE SET
  content = excluded.content,
  embedding_model = excluded.embedding_model,
  embedding_dimension = excluded.embedding_dimension,
  embedded_at = excluded.embedded_at
RETURNING id;
