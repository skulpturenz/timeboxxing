-- name: SearchTransitionEventDocuments :many
WITH matches AS (
  SELECT transition_event_document_id, distance
  FROM transition_event_document_float32_embeddings
  WHERE embedding MATCH ?
    AND k = ?
)
SELECT
  transition_event_documents.transition_event_id,
  transition_event_documents.content,
  matches.distance
FROM matches
JOIN transition_event_documents ON transition_event_documents.id = matches.transition_event_document_id
ORDER BY matches.distance;
