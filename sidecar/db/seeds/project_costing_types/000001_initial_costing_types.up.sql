INSERT INTO project_costing_types (id, label)
VALUES
  (1, 'hourly'),
  (2, 'deliverable')
ON CONFLICT(id) DO UPDATE SET
  label = excluded.label;
