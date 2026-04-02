-- piko.name: CreateTask
-- piko.command: exec
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- piko.name: UpdateTask
-- piko.command: exec
UPDATE tasks
SET
    status = ?, priority = ?, execute_at = ?, attempt = ?, last_error = ?, result = ?, payload = ?, config = ?, updated_at = ?
WHERE
    id = ?;

-- piko.name: FetchDueTasks
-- piko.command: many
-- ?1 as piko.slice(statuses)
SELECT
  id, workflow_id, executor, priority,
  payload, config,
  result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key
FROM tasks
WHERE
    status IN (?1)
    AND priority = ?2
    AND execute_at <= ?3
ORDER BY
  priority DESC,
  execute_at ASC,
  created_at ASC
LIMIT ?4;

-- piko.name: CreateTasksBatch
-- piko.command: batch
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status,
    execute_at, attempt, created_at, updated_at, deduplication_key
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- piko.name: MarkTasksAsProcessing
-- piko.command: exec
-- ?2 as piko.slice(ids)
UPDATE tasks
SET
  status = 'PROCESSING',
  updated_at = ?1
WHERE
  id IN (?2);
