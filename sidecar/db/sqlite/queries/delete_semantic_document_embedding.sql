-- name: DeleteSemanticDocumentEmbedding :exec
DELETE FROM semantic_document_embeddings
WHERE semantic_document_id = ?;
