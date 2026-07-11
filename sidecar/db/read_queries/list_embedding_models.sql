-- name: ListEmbeddingModels :many
SELECT id, openrouter_slug, ollama_slug, label
FROM embedding_models
ORDER BY id;
