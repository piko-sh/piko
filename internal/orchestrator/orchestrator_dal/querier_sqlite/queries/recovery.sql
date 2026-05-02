-- piko.query(name: RecoverStaleTasks, command: execrows)
UPDATE tasks
SET
    status = CASE WHEN attempt >= ? THEN 'FAILED' ELSE 'RETRYING' END,
    attempt = CASE WHEN attempt >= ? THEN attempt ELSE attempt + 1 END,
    last_error = ?,
    updated_at = ?,
    execute_at = ?
WHERE
    status = 'PROCESSING'
    AND updated_at < ?;

-- piko.query(name: GetStaleTasksForRecovery, command: many)
SELECT id, workflow_id, attempt FROM tasks
WHERE status = 'PROCESSING'
  AND updated_at < ?
  AND (recovery_node_id IS NULL OR recovery_expires_at < ?)
ORDER BY updated_at ASC
LIMIT ?;

-- piko.query(name: ClaimTaskForRecovery, command: execrows)
UPDATE tasks
SET recovery_node_id = ?, recovery_expires_at = ?
WHERE id = ? AND status = 'PROCESSING'
  AND (recovery_node_id IS NULL OR recovery_expires_at < ?);

-- piko.query(name: RecoverClaimedTasks, command: execrows)
UPDATE tasks
SET
    status = CASE WHEN attempt >= ? THEN 'FAILED' ELSE 'RETRYING' END,
    attempt = CASE WHEN attempt >= ? THEN attempt ELSE attempt + 1 END,
    last_error = ?,
    updated_at = ?,
    execute_at = ?,
    recovery_node_id = NULL,
    recovery_expires_at = NULL
WHERE
    recovery_node_id = ?
    AND status = 'PROCESSING';

-- piko.query(name: ReleaseRecoveryLeases, command: execrows)
UPDATE tasks
SET recovery_node_id = NULL, recovery_expires_at = NULL
WHERE recovery_node_id = ?;
