CREATE TABLE IF NOT EXISTS application_application_categories_map (
  id INTEGER PRIMARY KEY,
  application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  application_categories_id INTEGER NOT NULL REFERENCES application_categories(id) ON DELETE CASCADE,
  CONSTRAINT unique_application_category UNIQUE (application_id, application_categories_id)
);
