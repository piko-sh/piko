-- piko.name: CreateTask
-- piko.command: exec
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- piko.name: UpdateTask
-- piko.command: exec
UPDATE orchestrator_tasks
SET
    status = $1, priority = $2, execute_at = $3, attempt = $4, last_error = $5, result = $6, payload = $7, config = $8, updated_at = $9
WHERE
    id = $10;

-- piko.name: FetchDueTasks
-- piko.command: many
-- $1 as piko.slice(statuses)
SELECT
  id, workflow_id, executor, priority,
  payload,
  config,
  result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM orchestrator_tasks
WHERE
    status = ANY($1)
    AND priority = $2
    AND execute_at <= $3
ORDER BY
  priority DESC,
  execute_at ASC,
  created_at ASC
LIMIT $4;

-- piko.name: CreateTasksBatch
-- piko.command: batch
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status,
    execute_at, attempt, created_at, updated_at, deduplication_key
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- piko.name: MarkTasksAsProcessing
-- piko.command: exec
-- $1 as piko.slice(ids)
UPDATE orchestrator_tasks
SET
  status = 'PROCESSING',
  updated_at = $2
WHERE
  id = ANY($1);
