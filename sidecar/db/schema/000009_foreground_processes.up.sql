CREATE TABLE IF NOT EXISTS foreground_processes (
  id INTEGER PRIMARY KEY,
  application_id INTEGER REFERENCES applications(id),
  pid INTEGER,
  created_at_utc TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  CONSTRAINT unique_created_at_utc UNIQUE (created_at_utc)
);
