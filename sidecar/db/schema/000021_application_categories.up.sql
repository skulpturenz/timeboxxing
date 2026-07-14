CREATE TABLE IF NOT EXISTS application_categories (
  id INTEGER PRIMARY KEY,
  category_id INTEGER,
  code TEXT NOT NULL,
  label TEXT NOT NULL
);
