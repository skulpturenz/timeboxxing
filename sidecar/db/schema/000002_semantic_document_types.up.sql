CREATE TABLE IF NOT EXISTS semantic_document_types (
  id INTEGER PRIMARY KEY,
  code TEXT,
  CONSTRAINT unique_code UNIQUE (code)
);
