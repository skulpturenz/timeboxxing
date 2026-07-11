INSERT INTO models (id, semantic, embedding, openrouter_slug, ollama_slug, label)
VALUES
  (1, 0, 1, 'qwen/qwen3-embedding-8b', 'qwen3-embedding:8b', 'Qwen3 Embedding 8B'),
  (2, 0, 1, 'qwen/qwen3-embedding-4b', 'qwen3-embedding:4b', 'Qwen3 Embedding 4B'),
  (3, 0, 1, 'google/gemini-embedding-2-preview', '', 'Gemini Embedding 2 Preview'),
  (4, 1, 0, 'minimax/minimax-m3', '', 'MiniMax M3'),
  (5, 1, 0, 'deepseek/deepseek-v4-pro', '', 'DeepSeek V4 Pro'),
  (6, 1, 0, 'google/gemini-3-flash-preview', '', 'Gemini 3 Flash Preview'),
  (7, 1, 0, 'google/gemma-4-26b-a4b-it', 'gemma4:26b', 'Gemma 4 26B A4B IT')
ON CONFLICT(id) DO UPDATE SET
  semantic = excluded.semantic,
  embedding = excluded.embedding,
  openrouter_slug = excluded.openrouter_slug,
  ollama_slug = excluded.ollama_slug,
  label = excluded.label;
