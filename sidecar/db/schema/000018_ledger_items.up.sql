CREATE TABLE IF NOT EXISTS ledger_items (
  id INTEGER PRIMARY KEY,
  ledger_id INTEGER REFERENCES ledger(id),
  billable BOOLEAN NOT NULL DEFAULT 0,
  title TEXT NOT NULL,
  notes TEXT,
  started_at_utc TIMESTAMP,
  ended_at_utc TIMESTAMP,
  CHECK (started_at_utc IS NULL OR ended_at_utc IS NULL OR ended_at_utc > started_at_utc)
);
