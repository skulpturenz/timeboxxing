CREATE TABLE IF NOT EXISTS timesheets (
  id TEXT PRIMARY KEY,
  started_at TIMESTAMP NOT NULL,
  ended_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  CHECK (ended_at > started_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS timesheets_window_idx
  ON timesheets(started_at, ended_at);

CREATE TABLE IF NOT EXISTS timesheet_entries (
  id TEXT PRIMARY KEY,
  timesheet_id TEXT NOT NULL REFERENCES timesheets(id) ON DELETE CASCADE,
  project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  start_minute INTEGER NOT NULL CHECK (start_minute >= 0 AND start_minute < 1440),
  duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0 AND duration_minutes <= 720),
  billable BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS timesheet_entries_timesheet_idx
  ON timesheet_entries(timesheet_id, start_minute, created_at, id);

CREATE INDEX IF NOT EXISTS timesheet_entries_project_idx
  ON timesheet_entries(project_id);

CREATE TABLE IF NOT EXISTS timesheet_entry_usage_blocks (
  timesheet_entry_id TEXT NOT NULL REFERENCES timesheet_entries(id) ON DELETE CASCADE,
  usage_id TEXT NOT NULL,
  sort_order INTEGER NOT NULL CHECK (sort_order >= 0),
  PRIMARY KEY (timesheet_entry_id, usage_id)
);

CREATE INDEX IF NOT EXISTS timesheet_entry_usage_blocks_entry_order_idx
  ON timesheet_entry_usage_blocks(timesheet_entry_id, sort_order);
