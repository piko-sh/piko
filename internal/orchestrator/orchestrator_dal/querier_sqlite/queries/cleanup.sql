-- piko.query(name: CleanupOldResolvedReceipts, command: execrows)
DELETE FROM workflow_receipts
WHERE status = 'RESOLVED' AND resolved_at < ?;

-- piko.query(name: TimeoutStaleReceipts, command: execrows)
UPDATE workflow_receipts
SET status = 'TIMED_OUT', updated_at = ?
WHERE status = 'PENDING' AND created_at < ?;

-- piko.query(name: ListFailedTasks, command: many)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM tasks
WHERE status = 'FAILED'
ORDER BY updated_at DESC;
