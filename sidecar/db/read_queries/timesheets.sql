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

-- name: ListTimesheetEntriesInRange :many
SELECT
  timesheets.started_at,
  timesheets.ended_at,
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
WHERE timesheets.started_at >= ? AND timesheets.started_at < ?
ORDER BY timesheets.started_at, timesheet_entries.start_minute, timesheet_entries.created_at, timesheet_entries.id;

-- name: ListTimesheetEntryUsageBlocks :many
SELECT usage_id
FROM timesheet_entry_usage_blocks
WHERE timesheet_entry_id = ?
ORDER BY sort_order, usage_id;
