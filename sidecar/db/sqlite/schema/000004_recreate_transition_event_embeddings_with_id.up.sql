DROP TABLE IF EXISTS transition_event_document_float32_embeddings;
DELETE FROM transition_event_documents;

CREATE VIRTUAL TABLE IF NOT EXISTS transition_event_document_float32_embeddings USING vec0(
  id INTEGER PRIMARY KEY,
  transition_event_document_id INTEGER NOT NULL,
  embedding FLOAT[2048] distance_metric=cosine
);
