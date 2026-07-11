CREATE TABLE IF NOT EXISTS project_details (
  id INTEGER PRIMARY KEY,
  projects_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
  project_colors_id INTEGER REFERENCES project_colors(id),
  costing_type_id INTEGER REFERENCES project_costing_types(id),
  rate INTEGER,
  CONSTRAINT unique_projects_id UNIQUE (projects_id)
);
