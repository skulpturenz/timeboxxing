-- name: UpsertMaskedValue :one
INSERT INTO masked_values (masking_category, value, masked_value)
VALUES (?, ?, ?)
ON CONFLICT(masking_category, value) DO UPDATE SET
    value = excluded.value
RETURNING masked_value;
