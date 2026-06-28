-- name: CreateSemanticDocumentEmbedding :exec
INSERT INTO semantic_document_float32_embeddings (semantic_document_id, embedding)
VALUES (?, ?);
