-- piko.name: CleanupOldResolvedReceipts
-- piko.command: execrows
DELETE FROM orchestrator_workflow_receipts
WHERE status = 'RESOLVED' AND resolved_at < $1;

-- piko.name: TimeoutStaleReceipts
-- piko.command: execrows
UPDATE orchestrator_workflow_receipts
SET status = 'TIMED_OUT', updated_at = $1
WHERE status = 'PENDING' AND created_at < $2;

-- piko.name: ListFailedTasks
-- piko.command: many
SELECT
    id, workflow_id, executor, priority,
    payload,
    config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM orchestrator_tasks
WHERE status = 'FAILED'
ORDER BY updated_at DESC;
