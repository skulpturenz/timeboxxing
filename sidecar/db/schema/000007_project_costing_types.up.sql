CREATE TABLE IF NOT EXISTS project_costing_types (
  id INTEGER PRIMARY KEY,
  label TEXT,
  CONSTRAINT unique_label UNIQUE (label)
);
