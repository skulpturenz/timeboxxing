-- name: ListTimesheetEntries :many
SELECT
  ledger_items.id,
  project_costs.projects_id AS project_id,
  ledger_items.title,
  ledger_items.notes,
  ledger_items.billable,
  ledger_items.started_at_utc,
  ledger_items.ended_at_utc
FROM ledger_items
LEFT JOIN project_costs ON project_costs.ledger_items_id = ledger_items.id
WHERE ledger_items.started_at_utc >= ? AND ledger_items.started_at_utc < ?
ORDER BY ledger_items.started_at_utc, ledger_items.id;

-- name: ListTimesheetEntriesInRange :many
SELECT
  ledger_items.id,
  project_costs.projects_id AS project_id,
  ledger_items.title,
  ledger_items.notes,
  ledger_items.billable,
  ledger_items.started_at_utc,
  ledger_items.ended_at_utc
FROM ledger_items
LEFT JOIN project_costs ON project_costs.ledger_items_id = ledger_items.id
WHERE ledger_items.started_at_utc >= ? AND ledger_items.started_at_utc < ?
ORDER BY ledger_items.started_at_utc, ledger_items.id;

-- name: ListTimesheetEntryUsageBlocks :many
SELECT timeline_id
FROM ledger_item_timeline_entries
WHERE ledger_items_id = ?
ORDER BY timeline_id;
