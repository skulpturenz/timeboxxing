-- name: GetApplicationCategories :many
SELECT
    application_application_categories_map.application_id,
    application_categories.code
FROM application_application_categories_map
JOIN application_categories
    ON application_categories.id = application_application_categories_map.application_categories_id
WHERE application_application_categories_map.application_id IN (sqlc.slice('applicationIds'))
ORDER BY
    application_application_categories_map.application_id ASC,
    application_categories.id ASC;
