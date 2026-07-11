CREATE TABLE IF NOT EXISTS timeline_embeddings (
  id INTEGER PRIMARY KEY,
  timeline_id INTEGER REFERENCES timeline(id) ON DELETE CASCADE,
  timeline_semantic_documents_id INTEGER REFERENCES timeline_semantic_documents(id) ON DELETE CASCADE,
  embedding_model_id INTEGER REFERENCES models(id),
  dimension INTEGER NOT NULL,
  embedding BLOB NOT NULL,
  CONSTRAINT unique_timeline_semantic_documents_id_embedding_model_id UNIQUE (timeline_semantic_documents_id, embedding_model_id)
);
