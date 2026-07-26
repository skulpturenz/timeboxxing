CREATE TABLE IF NOT EXISTS foreground_process_metadata (
  id INTEGER PRIMARY KEY,
  foreground_process_id INTEGER NOT NULL REFERENCES foreground_processes(id) ON DELETE CASCADE,
  browser BOOLEAN NOT NULL DEFAULT 0,
  browser_vendor TEXT,
  browser_category INTEGER NOT NULL REFERENCES application_categories(id) ON DELETE SET NULL,
  idle BOOLEAN NOT NULL DEFAULT 0,
  tab TEXT,
  cdp_url TEXT,
  latitude REAL,
  longitude REAL,
  public_ip TEXT,
  title_source TEXT,
  killed BOOLEAN NOT NULL DEFAULT 0,
  window_title TEXT,
  CONSTRAINT unique_foreground_process_id UNIQUE (foreground_process_id)
);
