INSERT INTO transition_event_reasons (id, reason)
VALUES
  (1, 'focus_change'),
  (2, 'idle'),
  (3, 'return_from_idle'),
  (4, 'tab_change'),
  (5, 'shutdown'),
  (6, 'start')
ON CONFLICT(id) DO UPDATE SET reason = excluded.reason;
