-- name: UpsertApplicationCategoryMap :exec
INSERT INTO application_application_categories_map (application_id, application_categories_id)
VALUES (?, ?)
ON CONFLICT(application_id, application_categories_id) DO NOTHING;
