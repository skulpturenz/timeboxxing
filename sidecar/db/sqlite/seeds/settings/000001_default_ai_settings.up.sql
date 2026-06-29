INSERT INTO ai_settings (
  id,
  provider,
  openrouter_base_url,
  ollama_base_url,
  embedding_model_id,
  semantic_model_id,
  updated_at
)
VALUES (
  1,
  'openrouter',
  'https://openrouter.ai/api/v1',
  'http://127.0.0.1:11434',
  1,
  1,
  strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
)
ON CONFLICT(id) DO NOTHING;
