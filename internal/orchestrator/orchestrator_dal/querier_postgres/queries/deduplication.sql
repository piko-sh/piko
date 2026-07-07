-- piko.query(name: CreateTaskWithDedup, command: execrows)
INSERT INTO orchestrator_tasks (
    id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at, deduplication_key
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (deduplication_key)
    WHERE deduplication_key IS NOT NULL
    AND status IN ('SCHEDULED', 'PENDING', 'PROCESSING', 'RETRYING')
DO NOTHING;
