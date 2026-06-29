-- name: ListMissingSemanticEventDocumentIDs :many
SELECT transition_events.id
FROM transition_events
LEFT JOIN semantic_documents
  ON semantic_documents.transition_event_id = transition_events.id
  AND semantic_documents.document_type = 'event'
LEFT JOIN semantic_document_float32_embeddings
  ON semantic_document_float32_embeddings.semantic_document_id = semantic_documents.id
  AND semantic_document_float32_embeddings.embedding_model = ?
WHERE semantic_documents.id IS NULL
   OR semantic_document_float32_embeddings.id IS NULL
ORDER BY transition_events.started_at DESC, transition_events.id DESC
LIMIT ?;
