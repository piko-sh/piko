-- piko.name: CleanupOldResolvedReceipts
-- piko.command: execrows
DELETE FROM workflow_receipts
WHERE status = 'RESOLVED' AND resolved_at < ?;

-- piko.name: TimeoutStaleReceipts
-- piko.command: execrows
UPDATE workflow_receipts
SET status = 'TIMED_OUT', updated_at = ?
WHERE status = 'PENDING' AND created_at < ?;

-- piko.name: ListFailedTasks
-- piko.command: many
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM tasks
WHERE status = 'FAILED'
ORDER BY updated_at DESC;
