CREATE TABLE IF NOT EXISTS foreground_process_metadata (
  id INTEGER PRIMARY KEY,
  foreground_process_id INTEGER NOT NULL REFERENCES foreground_processes(id) ON DELETE CASCADE,
  browser BOOLEAN NOT NULL DEFAULT 0,
  idle BOOLEAN NOT NULL DEFAULT 0,
  tab TEXT,
  cdp_url TEXT,
  latitude REAL,
  longitude REAL,
  public_ip TEXT,
  CONSTRAINT unique_foreground_process_id UNIQUE (foreground_process_id)
);
