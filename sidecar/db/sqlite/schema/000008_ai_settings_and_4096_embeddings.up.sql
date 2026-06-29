DROP TABLE IF EXISTS semantic_document_float32_embeddings;

DROP INDEX IF EXISTS semantic_documents_type_started_idx;
DROP INDEX IF EXISTS semantic_documents_transition_event_idx;

CREATE TABLE semantic_documents_new (
  id INTEGER PRIMARY KEY,
  document_key TEXT NOT NULL UNIQUE,
  document_type TEXT NOT NULL,
  transition_event_id INTEGER REFERENCES transition_events(id),
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  content TEXT NOT NULL
);

INSERT INTO semantic_documents_new (
  id,
  document_key,
  document_type,
  transition_event_id,
  started_at,
  ended_at,
  content
)
SELECT
  id,
  document_key,
  document_type,
  transition_event_id,
  started_at,
  ended_at,
  content
FROM semantic_documents;

DROP TABLE semantic_documents;
ALTER TABLE semantic_documents_new RENAME TO semantic_documents;

CREATE INDEX IF NOT EXISTS semantic_documents_type_started_idx
  ON semantic_documents(document_type, started_at);

CREATE INDEX IF NOT EXISTS semantic_documents_transition_event_idx
  ON semantic_documents(transition_event_id);

CREATE VIRTUAL TABLE IF NOT EXISTS semantic_document_float32_embeddings USING vec0(
  id INTEGER PRIMARY KEY,
  semantic_document_id INTEGER NOT NULL,
  embedding_model TEXT NOT NULL,
  embedding_dimension INTEGER NOT NULL,
  embedded_at TEXT,
  embedding FLOAT[4096] distance_metric=cosine
);

CREATE TABLE IF NOT EXISTS embedding_models (
  id INTEGER PRIMARY KEY NOT NULL,
  openrouter_slug TEXT NOT NULL,
  ollama_slug TEXT NOT NULL,
  label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS semantic_models (
  id INTEGER PRIMARY KEY NOT NULL,
  openrouter_slug TEXT NOT NULL,
  ollama_slug TEXT NOT NULL,
  label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ai_settings (
  id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
  provider TEXT NOT NULL CHECK (provider IN ('openrouter', 'ollama')),
  openrouter_base_url TEXT NOT NULL,
  ollama_base_url TEXT NOT NULL,
  embedding_model_id INTEGER NOT NULL REFERENCES embedding_models(id),
  semantic_model_id INTEGER NOT NULL REFERENCES semantic_models(id),
  updated_at TIMESTAMP NOT NULL
);
