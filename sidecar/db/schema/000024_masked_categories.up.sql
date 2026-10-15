CREATE TABLE IF NOT EXISTS masked_categories (
  id INTEGER PRIMARY KEY,
  application_categories_id INTEGER NOT NULL REFERENCES application_categories(id) ON DELETE CASCADE,
  masked_application_categories_id INTEGER NOT NULL REFERENCES application_categories(id) ON DELETE CASCADE,
  CONSTRAINT unique_masked_category_source UNIQUE (application_categories_id),
  CONSTRAINT unique_masked_category_target UNIQUE (masked_application_categories_id)
);
