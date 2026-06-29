INSERT INTO semantic_models (id, openrouter_slug, ollama_slug, label)
VALUES
  (1, 'minimax/minimax-m3', '', 'MiniMax M3'),
  (2, 'deepseek/deepseek-v4-pro', '', 'DeepSeek V4 Pro'),
  (3, 'google/gemini-3-flash-preview', '', 'Gemini 3 Flash Preview'),
  (4, 'google/gemma-4-26b-a4b-it', 'gemma4:26b', 'Gemma 4 26B A4B IT')
ON CONFLICT(id) DO UPDATE SET
  openrouter_slug = excluded.openrouter_slug,
  ollama_slug = excluded.ollama_slug,
  label = excluded.label;

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
  CURRENT_TIMESTAMP
)
ON CONFLICT(id) DO NOTHING;
