CREATE TABLE IF NOT EXISTS application_categories (
  id INTEGER PRIMARY KEY,
  category_id INTEGER NOT NULL,
  code TEXT NOT NULL,
  label TEXT NOT NULL,
  CONSTRAINT unique_category_id UNIQUE (category_id)
);
