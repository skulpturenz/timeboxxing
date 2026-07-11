INSERT INTO application_settings (
  id,
  model_provider_id,
  model_provider_base_url,
  embedding_model_id,
  semantic_model_id,
  release_channel
)
VALUES (
  1,
  1,
  'https://openrouter.ai/api/v1',
  1,
  4,
  1
)
ON CONFLICT(id) DO NOTHING;
