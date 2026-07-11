-- name: DeleteTimelineEmbedding :exec
DELETE FROM timeline_embeddings
WHERE timeline_semantic_documents_id = ?;
