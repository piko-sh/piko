-- piko.query(name: CountUnsettledForBuild, command: one)
SELECT COUNT(*) FROM orchestrator_tasks
WHERE build_tag = $1 AND background = FALSE
AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING');

-- piko.query(name: ListFailedForBuild, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM orchestrator_tasks
WHERE build_tag = $1 AND status = 'FAILED'
ORDER BY updated_at, id;
