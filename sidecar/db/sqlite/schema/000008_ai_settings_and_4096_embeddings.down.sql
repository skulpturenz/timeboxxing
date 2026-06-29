DROP TABLE IF EXISTS ai_settings;
DROP TABLE IF EXISTS semantic_models;
DROP TABLE IF EXISTS embedding_models;

DROP TABLE IF EXISTS semantic_document_float32_embeddings;

DROP INDEX IF EXISTS semantic_documents_type_started_idx;
DROP INDEX IF EXISTS semantic_documents_transition_event_idx;

CREATE TABLE semantic_documents_old (
  id INTEGER PRIMARY KEY,
  document_key TEXT NOT NULL UNIQUE,
  document_type TEXT NOT NULL,
  transition_event_id INTEGER REFERENCES transition_events(id),
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  content TEXT NOT NULL,
  embedding_model TEXT NOT NULL DEFAULT '',
  embedding_dimension INTEGER NOT NULL DEFAULT 0,
  embedded_at TIMESTAMP
);

INSERT INTO semantic_documents_old (
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
ALTER TABLE semantic_documents_old RENAME TO semantic_documents;

CREATE INDEX IF NOT EXISTS semantic_documents_type_started_idx
  ON semantic_documents(document_type, started_at);

CREATE INDEX IF NOT EXISTS semantic_documents_transition_event_idx
  ON semantic_documents(transition_event_id);

CREATE VIRTUAL TABLE IF NOT EXISTS semantic_document_float32_embeddings USING vec0(
  id INTEGER PRIMARY KEY,
  semantic_document_id INTEGER NOT NULL,
  embedding FLOAT[2048] distance_metric=cosine
);
