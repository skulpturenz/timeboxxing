-- name: UpsertApplicationSettings :exec
INSERT INTO application_settings (
  id,
  model_provider_id,
  model_provider_base_url,
  embedding_model_id,
  semantic_model_id,
  release_channel
)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  model_provider_id = excluded.model_provider_id,
  model_provider_base_url = excluded.model_provider_base_url,
  embedding_model_id = excluded.embedding_model_id,
  semantic_model_id = excluded.semantic_model_id,
  release_channel = COALESCE(excluded.release_channel, application_settings.release_channel);
