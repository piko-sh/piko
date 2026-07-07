-- piko.query(name: CheckDuplicateActiveTask, command: one)
SELECT EXISTS(
    SELECT 1 FROM tasks
    WHERE deduplication_key = ?
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING')
) AS has_duplicate;

-- piko.query(name: CreateTaskWithDedup, command: exec)
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at, deduplication_key
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);
