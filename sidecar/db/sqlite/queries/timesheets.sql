-- name: GetTimesheetByWindow :one
SELECT
  id,
  started_at,
  ended_at,
  created_at,
  updated_at
FROM timesheets
WHERE started_at = ? AND ended_at = ?
LIMIT 1;

-- name: CreateTimesheet :one
INSERT INTO timesheets (
  id,
  started_at,
  ended_at,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?)
RETURNING
  id,
  started_at,
  ended_at,
  created_at,
  updated_at;

-- name: ListTimesheetEntries :many
SELECT
  timesheet_entries.id,
  timesheet_entries.timesheet_id,
  timesheet_entries.project_id,
  timesheet_entries.title,
  timesheet_entries.notes,
  timesheet_entries.start_minute,
  timesheet_entries.duration_minutes,
  timesheet_entries.billable,
  timesheet_entries.created_at,
  timesheet_entries.updated_at
FROM timesheet_entries
JOIN timesheets ON timesheets.id = timesheet_entries.timesheet_id
WHERE timesheets.started_at = ? AND timesheets.ended_at = ?
ORDER BY timesheet_entries.start_minute, timesheet_entries.created_at, timesheet_entries.id;

-- name: CreateTimesheetEntry :one
INSERT INTO timesheet_entries (
  id,
  timesheet_id,
  project_id,
  title,
  notes,
  start_minute,
  duration_minutes,
  billable,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING
  id,
  timesheet_id,
  project_id,
  title,
  notes,
  start_minute,
  duration_minutes,
  billable,
  created_at,
  updated_at;

-- name: CreateTimesheetEntryUsageBlock :exec
INSERT INTO timesheet_entry_usage_blocks (
  timesheet_entry_id,
  usage_id,
  sort_order
)
VALUES (?, ?, ?);

-- name: ListTimesheetEntryUsageBlocks :many
SELECT usage_id
FROM timesheet_entry_usage_blocks
WHERE timesheet_entry_id = ?
ORDER BY sort_order, usage_id;

-- name: DeleteTimesheetEntry :exec
DELETE FROM timesheet_entries
WHERE id = ?;
