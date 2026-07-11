CREATE TABLE IF NOT EXISTS project_colors (
  id INTEGER PRIMARY KEY,
  color INTEGER NOT NULL,
  description TEXT NOT NULL,
  CONSTRAINT unique_color UNIQUE (color)
);
