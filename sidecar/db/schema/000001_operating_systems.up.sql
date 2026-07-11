CREATE TABLE IF NOT EXISTS operating_systems (
  id INTEGER PRIMARY KEY,
  code TEXT NOT NULL,
  label TEXT NOT NULL,
  CONSTRAINT unique_id_code UNIQUE (id, code)
);
