-- name: ListModels :many
SELECT id, semantic, embedding, openrouter_slug, ollama_slug, label
FROM models
ORDER BY id;
