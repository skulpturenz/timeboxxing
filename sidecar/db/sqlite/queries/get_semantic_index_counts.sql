-- name: GetSemanticIndexCounts :one
SELECT
  (SELECT COUNT(*) FROM transition_events) AS completed_event_count,
  (
    SELECT COUNT(*)
    FROM semantic_documents
    WHERE document_type = 'event'
  ) AS indexed_event_count,
  (
    SELECT COUNT(*)
    FROM semantic_documents
    JOIN semantic_document_float32_embeddings
      ON semantic_document_float32_embeddings.semantic_document_id = semantic_documents.id
    WHERE semantic_documents.document_type = 'event'
  ) AS embedded_event_count;
