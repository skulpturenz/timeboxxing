DELETE FROM ai_settings
WHERE id = 1;

DELETE FROM semantic_models
WHERE id IN (1, 2, 3, 4);
