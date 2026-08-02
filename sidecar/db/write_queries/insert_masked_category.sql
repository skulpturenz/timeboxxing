-- name: InsertMaskedCategory :exec
INSERT INTO masked_categories (application_categories_id, masked_application_categories_id)
VALUES (
    (SELECT application_categories.id FROM application_categories WHERE application_categories.category_id = sqlc.arg(category_id)),
    (SELECT application_categories.id FROM application_categories WHERE application_categories.category_id = sqlc.arg(masked_category_id))
)
ON CONFLICT DO NOTHING;
