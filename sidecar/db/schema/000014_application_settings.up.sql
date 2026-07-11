CREATE TABLE IF NOT EXISTS application_settings (
  id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
  model_provider_id INTEGER REFERENCES model_providers(id),
  model_provider_base_url TEXT,
  embedding_model_id INTEGER NOT NULL REFERENCES models(id),
  semantic_model_id INTEGER NOT NULL REFERENCES models(id),
  release_channel INTEGER REFERENCES release_channels(id)
);
