-- name: EnsureTimesheet :one
INSERT INTO timesheets (
  id,
  started_at,
  ended_at,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (started_at, ended_at) DO UPDATE SET updated_at = timesheets.updated_at
RETURNING
  id,
  started_at,
  ended_at,
  created_at,
  updated_at;

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

-- name: DeleteTimesheetEntry :exec
DELETE FROM timesheet_entries
WHERE id = ?;
