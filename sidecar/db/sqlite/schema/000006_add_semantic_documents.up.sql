CREATE TABLE IF NOT EXISTS semantic_documents (
  id INTEGER PRIMARY KEY,
  document_key TEXT NOT NULL UNIQUE,
  document_type TEXT NOT NULL,
  transition_event_id INTEGER REFERENCES transition_events(id),
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  content TEXT NOT NULL,
  embedding_model TEXT NOT NULL,
  embedding_dimension INTEGER NOT NULL,
  embedded_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS semantic_documents_type_started_idx
  ON semantic_documents(document_type, started_at);

CREATE INDEX IF NOT EXISTS semantic_documents_transition_event_idx
  ON semantic_documents(transition_event_id);

CREATE VIRTUAL TABLE IF NOT EXISTS semantic_document_float32_embeddings USING vec0(
  id INTEGER PRIMARY KEY,
  semantic_document_id INTEGER NOT NULL,
  embedding FLOAT[2048] distance_metric=cosine
);
