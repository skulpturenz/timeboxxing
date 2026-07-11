-- name: UpsertTimelineSemanticDocument :one
INSERT INTO timeline_semantic_documents (document_key, timeline_id, type, content)
VALUES (?, ?, ?, ?)
ON CONFLICT(document_key) DO UPDATE SET
  timeline_id = excluded.timeline_id,
  type = excluded.type,
  content = excluded.content
RETURNING id;
