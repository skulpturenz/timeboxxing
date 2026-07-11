INSERT INTO model_providers (id, label)
VALUES
  (1, 'OpenRouter'),
  (2, 'Ollama')
ON CONFLICT(id) DO UPDATE SET
  label = excluded.label;
