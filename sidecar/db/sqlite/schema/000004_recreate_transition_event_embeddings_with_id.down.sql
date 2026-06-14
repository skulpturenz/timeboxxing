DROP TABLE IF EXISTS transition_event_document_float32_embeddings;
DELETE FROM transition_event_documents;

CREATE VIRTUAL TABLE IF NOT EXISTS transition_event_document_float32_embeddings USING vec0(
  transition_event_document_id INTEGER PRIMARY KEY,
  embedding FLOAT[2048] distance_metric=cosine
);
