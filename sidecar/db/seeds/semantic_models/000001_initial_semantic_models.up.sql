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
