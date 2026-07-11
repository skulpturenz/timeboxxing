-- name: ListMissingSemanticEventDocumentIDs :many
SELECT timeline.id
FROM timeline
LEFT JOIN timeline_semantic_documents
  ON timeline_semantic_documents.timeline_id = timeline.id
  AND timeline_semantic_documents.type = 1
LEFT JOIN timeline_embeddings
  ON timeline_embeddings.timeline_semantic_documents_id = timeline_semantic_documents.id
  AND timeline_embeddings.embedding_model_id = ?
WHERE timeline_semantic_documents.id IS NULL
   OR timeline_embeddings.id IS NULL
ORDER BY timeline.id DESC
LIMIT ?;
