INSERT INTO embedding_models (id, openrouter_slug, ollama_slug, label)
VALUES
  (1, 'qwen/qwen3-embedding-8b', 'qwen3-embedding:8b', 'Qwen3 Embedding 8B'),
  (2, 'qwen/qwen3-embedding-4b', 'qwen3-embedding:4b', 'Qwen3 Embedding 4B'),
  (3, 'google/gemini-embedding-2-preview', '', 'Gemini Embedding 2 Preview')
ON CONFLICT(id) DO UPDATE SET
  openrouter_slug = excluded.openrouter_slug,
  ollama_slug = excluded.ollama_slug,
  label = excluded.label;
