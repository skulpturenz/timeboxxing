-- name: GetMaskedValues :many
SELECT
    masked_values.masking_category,
    masked_values.value,
    masked_values.masked_value
FROM masked_values
ORDER BY masked_values.id ASC;
