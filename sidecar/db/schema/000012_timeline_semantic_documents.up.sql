CREATE TABLE IF NOT EXISTS timeline_semantic_documents (
  id INTEGER PRIMARY KEY,
  timeline_id INTEGER REFERENCES timeline(id) ON DELETE CASCADE,
  type INTEGER REFERENCES semantic_document_types(id),
  content TEXT NOT NULL,
  document_key TEXT NOT NULL,
  CONSTRAINT unique_document_key UNIQUE (document_key)
);
