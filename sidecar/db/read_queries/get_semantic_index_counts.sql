-- name: GetSemanticIndexCounts :one
SELECT
  (SELECT COUNT(*) FROM timeline) AS completed_event_count,
  (
    SELECT COUNT(*)
    FROM timeline_semantic_documents
    WHERE type = 1
  ) AS indexed_event_count,
  (
    SELECT COUNT(*)
    FROM timeline_semantic_documents
    JOIN timeline_embeddings
      ON timeline_embeddings.timeline_semantic_documents_id = timeline_semantic_documents.id
    WHERE timeline_semantic_documents.type = 1
      AND timeline_embeddings.embedding_model_id = ?
  ) AS embedded_event_count;
