-- piko.query(name: CreateTask, command: exec)
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at,
    deduplication_key, build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_error, result
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- piko.query(name: UpdateTask, command: execrows)
UPDATE tasks
SET
    status = ?, priority = ?, execute_at = ?, attempt = ?, last_error = ?, result = ?, payload = ?, config = ?, updated_at = ?,
    build_tag = ?, is_fatal = ?, scheduled_execute_at = ?, background = ?, owner_instance = ?
WHERE
    id = ?;

-- piko.query(name: GetTasksByID, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT
    id, workflow_id, executor, priority,
    payload, config,
    result, status, execute_at, attempt, last_error, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_published_at,
    rerun_id, rerun_payload, rerun_config
FROM tasks
WHERE
  id IN (?1);

-- piko.query(name: CreateTasksBatch, command: batch)
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status,
    execute_at, attempt, created_at, updated_at, deduplication_key,
    build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_error, result
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
