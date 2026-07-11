-- name: UpsertSemanticDocument :one
INSERT INTO semantic_documents (
  document_key,
  document_type,
  transition_event_id,
  started_at,
  ended_at,
  content
)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(document_key) DO UPDATE SET
  document_type = excluded.document_type,
  transition_event_id = excluded.transition_event_id,
  started_at = excluded.started_at,
  ended_at = excluded.ended_at,
  content = excluded.content
RETURNING id;
