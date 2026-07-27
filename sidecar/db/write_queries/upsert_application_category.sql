-- name: UpsertApplicationCategory :one
INSERT INTO application_categories (category_id, code, label)
VALUES (?, ?, ?)
ON CONFLICT(category_id) DO UPDATE SET
  code = excluded.code,
  label = excluded.label
RETURNING id;
