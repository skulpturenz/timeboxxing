-- name: GetApplicationCategoryCodes :many
SELECT
    application_categories.category_id,
    application_categories.code
FROM application_categories
ORDER BY application_categories.category_id ASC;
