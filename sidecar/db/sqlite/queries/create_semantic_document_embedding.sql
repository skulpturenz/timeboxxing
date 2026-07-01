-- name: CreateSemanticDocumentEmbedding :exec
INSERT INTO semantic_document_embeddings (
  semantic_document_id,
  embedding_model,
  embedding_dimension,
  embedded_at,
  embedding
)
VALUES (?, ?, ?, ?, ?);
