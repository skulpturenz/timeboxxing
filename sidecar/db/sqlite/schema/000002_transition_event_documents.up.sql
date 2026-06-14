CREATE TABLE IF NOT EXISTS transition_event_documents (
  id INTEGER PRIMARY KEY,
  transition_event_id INTEGER NOT NULL UNIQUE REFERENCES transition_events(id),
  content TEXT NOT NULL,
  embedding_model TEXT NOT NULL,
  embedding_dimension INTEGER NOT NULL,
  embedded_at TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS transition_event_document_float32_embeddings USING vec0(
  transition_event_document_id INTEGER PRIMARY KEY,
  embedding FLOAT[2048] distance_metric=cosine
);
