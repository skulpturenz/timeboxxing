-- name: CreateTimelineEmbedding :exec
INSERT INTO timeline_embeddings (
  timeline_id,
  timeline_semantic_documents_id,
  embedding_model_id,
  dimension,
  embedding
)
VALUES (?, ?, ?, ?, ?);
