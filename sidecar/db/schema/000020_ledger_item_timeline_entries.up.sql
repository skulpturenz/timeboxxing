CREATE TABLE IF NOT EXISTS ledger_item_timeline_entries (
  id INTEGER PRIMARY KEY,
  ledger_items_id INTEGER REFERENCES ledger_items(id) ON DELETE CASCADE,
  timeline_id INTEGER REFERENCES timeline(id) ON DELETE CASCADE
);
