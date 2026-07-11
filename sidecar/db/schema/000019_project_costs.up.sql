CREATE TABLE IF NOT EXISTS project_costs (
  id INTEGER PRIMARY KEY,
  ledger_items_id INTEGER REFERENCES ledger_items(id) ON DELETE CASCADE,
  projects_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
  costing_type_id INTEGER REFERENCES project_costing_types(id),
  rate INTEGER
);
