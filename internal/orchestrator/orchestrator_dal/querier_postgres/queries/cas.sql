-- piko.query(name: ClaimTask, command: one, optional: true)
UPDATE orchestrator_tasks
SET status = 'PROCESSING', attempt = attempt + 1, updated_at = $1, owner_instance = $2
WHERE id = $3 AND attempt = $4
  AND (status = 'PENDING' OR (status IN ('RETRYING', 'SCHEDULED') AND execute_at <= $5))
RETURNING
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config;

-- piko.query(name: SettleTask, command: one, optional: true)
UPDATE orchestrator_tasks
SET status = $1, execute_at = $2, last_error = $3, result = $4, is_fatal = $5,
    scheduled_execute_at = $6, updated_at = $7, owner_instance = ''
WHERE id = $8 AND attempt = $9 AND status = 'PROCESSING'
RETURNING
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config;

-- piko.query(name: TransitionTask, command: one, optional: true)
-- $11 as piko.param(from_statuses, kind: slice)
UPDATE orchestrator_tasks
SET status = $1, execute_at = $2, last_error = $3, result = $4, is_fatal = $5,
    scheduled_execute_at = $6, updated_at = $7, owner_instance = $8
WHERE id = $9 AND attempt = $10 AND status IN ($11)
RETURNING
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config;

-- piko.query(name: ClearRerun, command: exec)
UPDATE orchestrator_tasks
SET rerun_id = NULL, rerun_payload = NULL, rerun_config = NULL
WHERE id = $1;

-- piko.query(name: RequestRerun, command: execrows)
UPDATE orchestrator_tasks
SET rerun_id = $1, rerun_payload = $2, rerun_config = $3
WHERE deduplication_key = $4 AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING');
