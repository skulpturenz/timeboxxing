-- name: DeleteSemanticDocumentEmbedding :exec
DELETE FROM semantic_document_float32_embeddings
WHERE semantic_document_id = ?;
