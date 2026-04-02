-- piko.name: RecoverStaleTasks
-- piko.command: execrows
UPDATE orchestrator_tasks
SET
    status = CASE WHEN attempt >= $1 THEN 'FAILED' ELSE 'RETRYING' END,
    attempt = CASE WHEN attempt >= $1 THEN attempt ELSE attempt + 1 END,
    last_error = $2,
    updated_at = $3,
    execute_at = $3
WHERE
    status = 'PROCESSING'
    AND updated_at < $4;

-- piko.name: ClaimStaleTasksForRecovery
-- piko.command: many
WITH claimable AS (
    SELECT orchestrator_tasks.id FROM orchestrator_tasks
    WHERE orchestrator_tasks.status = 'PROCESSING'
      AND orchestrator_tasks.updated_at < $1
      AND (orchestrator_tasks.recovery_node_id IS NULL OR orchestrator_tasks.recovery_expires_at < $2)
    ORDER BY orchestrator_tasks.updated_at ASC
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE orchestrator_tasks
SET recovery_node_id = $4,
    recovery_expires_at = $5
FROM claimable
WHERE orchestrator_tasks.id = claimable.id
RETURNING orchestrator_tasks.id, orchestrator_tasks.workflow_id, orchestrator_tasks.attempt;

-- piko.name: RecoverClaimedTasks
-- piko.command: execrows
UPDATE orchestrator_tasks
SET
    status = CASE WHEN attempt >= $1 THEN 'FAILED' ELSE 'RETRYING' END,
    attempt = CASE WHEN attempt >= $1 THEN attempt ELSE attempt + 1 END,
    last_error = $2,
    updated_at = $3,
    execute_at = $3,
    recovery_node_id = NULL,
    recovery_expires_at = NULL
WHERE
    recovery_node_id = $4
    AND status = 'PROCESSING';

-- piko.name: ReleaseRecoveryLeases
-- piko.command: execrows
UPDATE orchestrator_tasks
SET recovery_node_id = NULL, recovery_expires_at = NULL
WHERE recovery_node_id = $1;
