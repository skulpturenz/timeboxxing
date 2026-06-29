-- name: SearchSemanticDocuments :many
WITH matches AS (
  SELECT semantic_document_id, distance
  FROM semantic_document_float32_embeddings
  WHERE embedding MATCH ?
    AND embedding_model = ?
    AND k = ?
)
SELECT
  semantic_documents.id,
  semantic_documents.document_key,
  semantic_documents.document_type,
  semantic_documents.transition_event_id,
  semantic_documents.started_at,
  semantic_documents.ended_at,
  semantic_documents.content,
  matches.distance
FROM matches
JOIN semantic_documents ON semantic_documents.id = matches.semantic_document_id
ORDER BY matches.distance;
