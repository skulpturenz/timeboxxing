-- name: GetAISettings :one
SELECT
  ai_settings.id,
  ai_settings.provider,
  ai_settings.openrouter_base_url,
  ai_settings.ollama_base_url,
  ai_settings.embedding_model_id,
  ai_settings.semantic_model_id,
  ai_settings.updated_at,
  embedding_models.openrouter_slug AS embedding_openrouter_slug,
  embedding_models.ollama_slug AS embedding_ollama_slug,
  embedding_models.label AS embedding_label,
  semantic_models.openrouter_slug AS semantic_openrouter_slug,
  semantic_models.ollama_slug AS semantic_ollama_slug,
  semantic_models.label AS semantic_label
FROM ai_settings
JOIN embedding_models ON embedding_models.id = ai_settings.embedding_model_id
JOIN semantic_models ON semantic_models.id = ai_settings.semantic_model_id
WHERE ai_settings.id = 1;
