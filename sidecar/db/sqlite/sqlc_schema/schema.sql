CREATE TABLE applications (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  platform_identifier TEXT,
  path TEXT
);

CREATE TABLE transition_event_reasons (
  id INTEGER PRIMARY KEY NOT NULL,
  reason TEXT NOT NULL UNIQUE
);

CREATE TABLE transition_events (
  id INTEGER PRIMARY KEY,
  application_id INTEGER REFERENCES applications(id),
  transition_reason_id INTEGER NOT NULL REFERENCES transition_event_reasons(id),
  started_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  ended_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE transition_event_metadata (
  id INTEGER PRIMARY KEY,
  transition_event_id INTEGER NOT NULL REFERENCES transition_events(id),
  browser BOOLEAN NOT NULL DEFAULT false,
  tab TEXT,
  idle BOOLEAN NOT NULL DEFAULT false,
  cdp_url TEXT,
  pid INTEGER
);

CREATE TABLE semantic_documents (
  id INTEGER PRIMARY KEY,
  document_key TEXT NOT NULL UNIQUE,
  document_type TEXT NOT NULL,
  transition_event_id INTEGER REFERENCES transition_events(id),
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  content TEXT NOT NULL
);

CREATE INDEX semantic_documents_type_started_idx
  ON semantic_documents(document_type, started_at);

CREATE INDEX semantic_documents_transition_event_idx
  ON semantic_documents(transition_event_id);

CREATE TABLE semantic_document_float32_embeddings (
  id INTEGER PRIMARY KEY,
  semantic_document_id INTEGER NOT NULL,
  embedding_model TEXT NOT NULL,
  embedding_dimension INTEGER NOT NULL,
  embedded_at TIMESTAMP,
  embedding TEXT NOT NULL,
  k INTEGER,
  distance REAL
);

CREATE TABLE embedding_models (
  id INTEGER PRIMARY KEY NOT NULL,
  openrouter_slug TEXT NOT NULL,
  ollama_slug TEXT NOT NULL,
  label TEXT NOT NULL
);

CREATE TABLE semantic_models (
  id INTEGER PRIMARY KEY NOT NULL,
  openrouter_slug TEXT NOT NULL,
  ollama_slug TEXT NOT NULL,
  label TEXT NOT NULL
);

CREATE TABLE ai_settings (
  id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
  provider TEXT NOT NULL CHECK (provider IN ('openrouter', 'ollama')),
  openrouter_base_url TEXT NOT NULL,
  ollama_base_url TEXT NOT NULL,
  embedding_model_id INTEGER NOT NULL REFERENCES embedding_models(id),
  semantic_model_id INTEGER NOT NULL REFERENCES semantic_models(id),
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  color_argb INTEGER NOT NULL CHECK (color_argb BETWEEN 0 AND 4294967295),
  client TEXT NOT NULL DEFAULT '',
  hourly_rate_cents INTEGER NOT NULL DEFAULT 0 CHECK (hourly_rate_cents >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE UNIQUE INDEX projects_name_nocase_idx
  ON projects(name COLLATE NOCASE);

CREATE TABLE timesheets (
  id TEXT PRIMARY KEY,
  started_at TIMESTAMP NOT NULL,
  ended_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  CHECK (ended_at > started_at)
);

CREATE UNIQUE INDEX timesheets_window_idx
  ON timesheets(started_at, ended_at);

CREATE TABLE timesheet_entries (
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

CREATE INDEX timesheet_entries_timesheet_idx
  ON timesheet_entries(timesheet_id, start_minute, created_at, id);

CREATE INDEX timesheet_entries_project_idx
  ON timesheet_entries(project_id);

CREATE TABLE timesheet_entry_usage_blocks (
  timesheet_entry_id TEXT NOT NULL REFERENCES timesheet_entries(id) ON DELETE CASCADE,
  usage_id TEXT NOT NULL,
  sort_order INTEGER NOT NULL CHECK (sort_order >= 0),
  PRIMARY KEY (timesheet_entry_id, usage_id)
);

CREATE INDEX timesheet_entry_usage_blocks_entry_order_idx
  ON timesheet_entry_usage_blocks(timesheet_entry_id, sort_order);
