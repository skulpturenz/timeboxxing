-- name: DeleteTransitionEventDocumentEmbedding :exec
DELETE FROM transition_event_document_float32_embeddings
WHERE transition_event_document_id = ?;
