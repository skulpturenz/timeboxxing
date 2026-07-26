-- name: GetUnindexedForegroundProcesses :many
SELECT
    foreground_processes.id,
    applications.name AS application_name,
    applications.identifier AS application_identifier,
    foreground_processes.application_id,
    foreground_processes.pid,
    foreground_processes.created_at_utc,
    foreground_process_metadata.browser,
    foreground_process_metadata.browser_vendor,
    bc.code AS browser_category,
    foreground_process_metadata.idle,
    foreground_process_metadata.killed,
    foreground_process_metadata.tab,
    foreground_process_metadata.cdp_url,
    foreground_process_metadata.latitude,
    foreground_process_metadata.longitude,
    foreground_process_metadata.public_ip,
    foreground_process_metadata.title_source,
    foreground_process_metadata.window_title
FROM foreground_processes
JOIN applications ON applications.id = foreground_processes.application_id
LEFT JOIN application_categories bc ON bc.id = foreground_process_metadata.browser_category
JOIN foreground_process_metadata ON foreground_process_metadata.foreground_process_id = foreground_processes.id
WHERE       foreground_processes.id NOT IN (SELECT timeline.initial_foreground_process_id AS foreground_process_id
                                          FROM timeline
                                          JOIN timeline_semantic_documents ON timeline_semantic_documents.timeline_id = timeline.id
                                          WHERE timeline.initial_foreground_process_id IS NOT NULL
                                          UNION ALL
                                          SELECT timeline.end_foreground_process_id AS foreground_process_id
                                          FROM timeline
                                          JOIN timeline_semantic_documents ON timeline_semantic_documents.timeline_id = timeline.id
                                          WHERE timeline.end_foreground_process_id IS NOT NULL)
      AND   foreground_processes.id > @foregroundProcessId
LIMIT @pageSize;
