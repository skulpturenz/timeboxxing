-- name: EnsureLedger :exec
INSERT INTO ledger (id)
VALUES (1)
ON CONFLICT(id) DO NOTHING;

-- name: CreateLedgerItem :one
INSERT INTO ledger_items (ledger_id, billable, title, notes, started_at_utc, ended_at_utc)
VALUES (1, ?, ?, ?, ?, ?)
RETURNING
  id,
  ledger_id,
  billable,
  title,
  notes,
  started_at_utc,
  ended_at_utc;

-- name: CreateProjectCost :exec
INSERT INTO project_costs (ledger_items_id, projects_id, costing_type_id, rate)
VALUES (?, ?, ?, ?);

-- name: CreateLedgerItemTimelineEntry :exec
INSERT INTO ledger_item_timeline_entries (ledger_items_id, timeline_id)
VALUES (?, ?);

-- name: DeleteLedgerItem :exec
DELETE FROM ledger_items WHERE id = ?;
