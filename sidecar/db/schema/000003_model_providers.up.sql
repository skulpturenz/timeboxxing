CREATE TABLE IF NOT EXISTS model_providers (
  id INTEGER PRIMARY KEY,
  label TEXT NOT NULL,
  CONSTRAINT unique_label UNIQUE (label)
);
