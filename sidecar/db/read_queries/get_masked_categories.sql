-- name: GetMaskedCategories :many
SELECT
    source.category_id AS category_id,
    target.category_id AS masked_category_id
FROM masked_categories
JOIN application_categories AS source
    ON source.id = masked_categories.application_categories_id
JOIN application_categories AS target
    ON target.id = masked_categories.masked_application_categories_id
ORDER BY source.category_id ASC;
