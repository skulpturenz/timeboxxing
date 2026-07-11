INSERT INTO semantic_document_types (id, code)
VALUES
  (1, 'event'),
  (2, 'day_summary'),
  (3, 'app_day_summary'),
  (4, 'time_block_summary')
ON CONFLICT(id) DO UPDATE SET
  code = excluded.code;
