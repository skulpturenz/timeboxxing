CREATE TABLE IF NOT EXISTS applications (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  operating_system_id INTEGER REFERENCES operating_systems(id),
  path TEXT,
  CONSTRAINT unique_name UNIQUE (name)
);
