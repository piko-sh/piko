-- piko.query(name: CreateTaskWithDedup, command: execrows)
INSERT INTO tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at,
    deduplication_key, build_tag, is_fatal, scheduled_execute_at, background, owner_instance, last_error, result
)
SELECT ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19
WHERE NOT EXISTS (
    SELECT 1 FROM tasks
    WHERE deduplication_key = ?12
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING')
);
