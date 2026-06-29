-- name: UpsertAISettings :exec
INSERT INTO ai_settings (
  id,
  provider,
  openrouter_base_url,
  ollama_base_url,
  embedding_model_id,
  semantic_model_id,
  updated_at
)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  provider = excluded.provider,
  openrouter_base_url = excluded.openrouter_base_url,
  ollama_base_url = excluded.ollama_base_url,
  embedding_model_id = excluded.embedding_model_id,
  semantic_model_id = excluded.semantic_model_id,
  updated_at = excluded.updated_at;
