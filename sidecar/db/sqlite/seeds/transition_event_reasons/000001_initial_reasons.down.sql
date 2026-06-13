DELETE FROM transition_event_reasons
WHERE reason IN (
  'focus_change',
  'idle',
  'return_from_idle',
  'tab_change',
  'shutdown',
  'start'
);
