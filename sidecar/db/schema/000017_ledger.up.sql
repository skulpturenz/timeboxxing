CREATE TABLE IF NOT EXISTS ledger (
  id INTEGER PRIMARY KEY,
  created_at_utc TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at_utc TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
