CREATE TABLE IF NOT EXISTS masked_values (
  id INTEGER PRIMARY KEY,
  masking_category TEXT NOT NULL,
  value TEXT NOT NULL,
  masked_value TEXT NOT NULL,
  CONSTRAINT unique_masked_value_source UNIQUE (masking_category, value),
  CONSTRAINT unique_masked_value_token UNIQUE (masking_category, masked_value)
);
