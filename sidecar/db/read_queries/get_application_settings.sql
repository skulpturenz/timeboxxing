-- name: GetApplicationSettings :one
SELECT
  application_settings.id,
  application_settings.model_provider_id,
  model_providers.label AS model_provider_label,
  application_settings.model_provider_base_url,
  application_settings.embedding_model_id,
  application_settings.semantic_model_id,
  application_settings.release_channel,
  embedding_models.openrouter_slug AS embedding_openrouter_slug,
  embedding_models.ollama_slug AS embedding_ollama_slug,
  embedding_models.label AS embedding_label,
  semantic_models.openrouter_slug AS semantic_openrouter_slug,
  semantic_models.ollama_slug AS semantic_ollama_slug,
  semantic_models.label AS semantic_label
FROM application_settings
LEFT JOIN model_providers ON model_providers.id = application_settings.model_provider_id
JOIN models AS embedding_models ON embedding_models.id = application_settings.embedding_model_id
JOIN models AS semantic_models ON semantic_models.id = application_settings.semantic_model_id
WHERE application_settings.id = 1;
