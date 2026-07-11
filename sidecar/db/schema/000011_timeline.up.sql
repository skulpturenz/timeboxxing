CREATE TABLE IF NOT EXISTS timeline (
  id INTEGER PRIMARY KEY,
  initial_foreground_process_id INTEGER REFERENCES foreground_processes(id) ON DELETE CASCADE,
  end_foreground_process_id INTEGER REFERENCES foreground_processes(id) ON DELETE CASCADE
);
