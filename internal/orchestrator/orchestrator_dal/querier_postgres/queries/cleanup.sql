-- piko.query(name: CleanupOldResolvedReceipts, command: execrows)
DELETE FROM orchestrator_workflow_receipts
WHERE id IN (SELECT id FROM orchestrator_workflow_receipts WHERE status IN ('RESOLVED', 'TIMED_OUT') AND COALESCE(resolved_at, updated_at) < $1 ORDER BY COALESCE(resolved_at, updated_at), id LIMIT 100);

-- piko.query(name: TimeoutStaleReceipts, command: execrows)
UPDATE orchestrator_workflow_receipts
SET status = 'TIMED_OUT', updated_at = $1
WHERE id IN (SELECT id FROM orchestrator_workflow_receipts WHERE status = 'PENDING' AND created_at < $2 ORDER BY created_at, id LIMIT 500);

-- piko.query(name: ListFailedTasks, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM orchestrator_tasks
WHERE status = 'FAILED'
ORDER BY updated_at DESC;
