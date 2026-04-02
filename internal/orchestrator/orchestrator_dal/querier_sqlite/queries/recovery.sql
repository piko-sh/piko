-- piko.name: RecoverStaleTasks
-- piko.command: execrows
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

-- piko.name: GetStaleTasksForRecovery
-- piko.command: many
SELECT id, workflow_id, attempt FROM tasks
WHERE status = 'PROCESSING'
  AND updated_at < ?
  AND (recovery_node_id IS NULL OR recovery_expires_at < ?)
ORDER BY updated_at ASC
LIMIT ?;

-- piko.name: ClaimTaskForRecovery
-- piko.command: execrows
UPDATE tasks
SET recovery_node_id = ?, recovery_expires_at = ?
WHERE id = ? AND status = 'PROCESSING'
  AND (recovery_node_id IS NULL OR recovery_expires_at < ?);

-- piko.name: RecoverClaimedTasks
-- piko.command: execrows
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

-- piko.name: ReleaseRecoveryLeases
-- piko.command: execrows
UPDATE tasks
SET recovery_node_id = NULL, recovery_expires_at = NULL
WHERE recovery_node_id = ?;
