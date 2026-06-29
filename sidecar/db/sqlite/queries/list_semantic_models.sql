-- name: ListSemanticModels :many
SELECT id, openrouter_slug, ollama_slug, label
FROM semantic_models
ORDER BY id;
