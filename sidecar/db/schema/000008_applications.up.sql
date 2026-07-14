CREATE TABLE IF NOT EXISTS applications (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  identifier TEXT,
  operating_system_id INTEGER REFERENCES operating_systems(id),
  path TEXT,
  CONSTRAINT unique_identifier_operating_system_id UNIQUE (identifier, operating_system_id)
);
