-- piko.query(name: CreateTask, command: exec)
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at,
    deduplication_key, build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_error, result
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
);

-- piko.query(name: UpdateTask, command: execrows)
UPDATE orchestrator_tasks
SET
    status = $1, priority = $2, execute_at = $3, attempt = $4, last_error = $5, result = $6, payload = $7, config = $8, updated_at = $9,
    build_tag = $10, is_fatal = $11, scheduled_execute_at = $12, background = $13, owner_instance = $14
WHERE
    id = $15;

-- piko.query(name: GetTasksByID, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM orchestrator_tasks
WHERE
  id IN ($1);

-- piko.query(name: CreateTasksBatch, command: batch)
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status,
    execute_at, attempt, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_error, result
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19);
