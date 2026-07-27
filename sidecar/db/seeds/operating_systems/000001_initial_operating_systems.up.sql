INSERT INTO operating_systems (id, code, label)
VALUES
  (1, 'darwin', 'macOS'),
  (2, 'windows', 'Windows')
ON CONFLICT(id) DO UPDATE SET
  code = excluded.code,
  label = excluded.label;
