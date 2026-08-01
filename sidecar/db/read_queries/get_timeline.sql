-- name: GetTimeline :many
SELECT
    timeline.id,
    -- initial
    timeline.initial_foreground_process_id AS initial_fp_id,
    initial_app.name AS initial_application_name,
    initial_app.identifier AS initial_application_identifier,
    initial_app.path AS initial_application_path,
    -- app category not selected, use `GetApplicationCategories` and merge
    initial.application_id AS initial_application_id,
    initial.pid AS initial_pid,
    initial.created_at_utc AS initial_created_at_utc,
    initial_fpm.browser AS initial_browser,
    initial_fpm.browser_vendor AS initial_browser_vendor,
    initial_bc.code AS initial_browser_category,
    initial_fpm.idle AS initial_idle,
    initial_fpm.killed AS initial_killed,
    initial_fpm.tab AS initial_tab,
    initial_fpm.cdp_url AS initial_cdp_url,
    initial_fpm.latitude AS initial_latitude,
    initial_fpm.longitude AS initial_longitude,
    initial_fpm.public_ip AS initial_public_ip,
    initial_fpm.title_source AS initial_title_source,
    initial_fpm.window_title AS initial_window_title,
    -- final
    timeline.end_foreground_process_id AS final_fp_id,
    final_app.name AS final_application_name,
    final_app.identifier AS final_application_identifier,
    final_app.path AS final_application_path,
    -- app category not selected, use `GetApplicationCategories` and merge
    final.application_id AS final_application_id,
    final.pid AS final_pid,
    final.created_at_utc AS final_created_at_utc,
    final_fpm.browser AS final_browser,
    final_fpm.browser_vendor AS final_browser_vendor,
    final_bc.code AS final_browser_category,
    final_fpm.idle AS final_idle,
    final_fpm.killed AS final_killed,
    final_fpm.tab AS final_tab,
    final_fpm.cdp_url AS final_cdp_url,
    final_fpm.latitude AS final_latitude,
    final_fpm.longitude AS final_longitude,
    final_fpm.public_ip AS final_public_ip,
    final_fpm.title_source AS final_title_source,
    final_fpm.window_title AS final_window_title
FROM timeline
LEFT JOIN foreground_processes initial ON initial.id = timeline.initial_foreground_process_id
LEFT JOIN foreground_processes final ON final.id = timeline.end_foreground_process_id -- absent until the entry is closed
LEFT JOIN foreground_process_metadata initial_fpm ON initial_fpm.foreground_process_id = initial.id
LEFT JOIN foreground_process_metadata final_fpm ON final_fpm.foreground_process_id = final.id
LEFT JOIN applications initial_app ON initial_app.id = initial.application_id -- an idle observation has no application
LEFT JOIN applications final_app ON final_app.id = final.application_id
LEFT JOIN application_categories initial_bc ON initial_bc.id = initial_fpm.browser_category -- only browsers are categorized
LEFT JOIN application_categories final_bc ON final_bc.id = final_fpm.browser_category
WHERE   timeline.id > @timelineId
    AND timeline.initial_foreground_process_id IS NOT NULL
    AND (   timeline.end_foreground_process_id IS NULL -- active app. if killed or idle it should be non null
        OR  unixepoch(final.created_at_utc) - unixepoch(initial.created_at_utc)
                >= CAST(sqlc.arg('min_duration_seconds') AS INTEGER)
        )
    AND (   timeline.end_foreground_process_id IS NULL
        OR  (   CAST(sqlc.narg('started_at') AS TIMESTAMP) IS NULL
            OR  unixepoch(final.created_at_utc, 'subsec')
                    > unixepoch(sqlc.narg('started_at'), 'subsec'))
        )
    AND (   CAST(sqlc.narg('ended_at') AS TIMESTAMP) IS NULL
        OR  unixepoch(initial.created_at_utc, 'subsec')
                < unixepoch(sqlc.narg('ended_at'), 'subsec'))
ORDER BY timeline.id ASC
LIMIT @pageSize;
