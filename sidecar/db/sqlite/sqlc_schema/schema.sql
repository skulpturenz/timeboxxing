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
  started_at TIMESTAMP NOT NULL,
  ended_at TIMESTAMP NOT NULL
);

CREATE TABLE transition_event_metadata (
  id INTEGER PRIMARY KEY,
  transition_event_id INTEGER NOT NULL REFERENCES transition_events(id),
  browser BOOLEAN NOT NULL DEFAULT false,
  tab TEXT,
  idle BOOLEAN NOT NULL DEFAULT false,
  cdp_url TEXT
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
