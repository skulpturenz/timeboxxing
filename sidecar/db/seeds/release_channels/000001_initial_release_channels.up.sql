INSERT INTO release_channels (id, label)
VALUES
  (1, 'Stable'),
  (2, 'Beta'),
  (3, 'Alpha')
ON CONFLICT(id) DO UPDATE SET
  label = excluded.label;
