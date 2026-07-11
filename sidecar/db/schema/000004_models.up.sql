CREATE TABLE IF NOT EXISTS models (
  id INTEGER PRIMARY KEY,
  semantic BOOLEAN NOT NULL DEFAULT 0,
  embedding BOOLEAN NOT NULL DEFAULT 0,
  openrouter_slug TEXT,
  ollama_slug TEXT,
  label TEXT NOT NULL
);
