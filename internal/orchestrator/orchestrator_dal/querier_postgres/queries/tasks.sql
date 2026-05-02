-- piko.query(name: CreateTask, command: exec)
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- piko.query(name: UpdateTask, command: exec)
UPDATE orchestrator_tasks
SET
    status = $1, priority = $2, execute_at = $3, attempt = $4, last_error = $5, result = $6, payload = $7, config = $8, updated_at = $9
WHERE
    id = $10;

-- piko.query(name: FetchDueTasks, command: many)
-- $1 as piko.param(statuses, kind: slice)
SELECT
  id, workflow_id, executor, priority,
  payload,
  config,
  result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM orchestrator_tasks
WHERE
    status IN ($1)
    AND priority = $2
    AND execute_at <= $3
ORDER BY
  priority DESC,
  execute_at ASC,
  created_at ASC
LIMIT $4;

-- piko.query(name: CreateTasksBatch, command: batch)
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status,
    execute_at, attempt, created_at, updated_at, deduplication_key
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- piko.query(name: MarkTasksAsProcessing, command: exec)
-- $1 as piko.param(ids, kind: slice)
UPDATE orchestrator_tasks
SET
  status = 'PROCESSING',
  updated_at = $2
WHERE
  id IN ($1);
