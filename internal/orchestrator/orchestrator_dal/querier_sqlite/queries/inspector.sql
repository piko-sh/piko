-- piko.name: ListTaskStatusCounts
-- piko.command: many
SELECT status, COUNT(*) AS task_count
FROM tasks
GROUP BY status;

-- piko.name: ListRecentTasks
-- piko.command: many
SELECT
    id, workflow_id, executor, status, priority, attempt,
    last_error, created_at, updated_at
FROM tasks
ORDER BY updated_at DESC
LIMIT ?;

-- piko.name: ListWorkflowSummary
-- piko.command: many
SELECT
    workflow_id,
    COUNT(*) AS task_count,
    SUM(CASE WHEN status = 'COMPLETE' THEN 1 ELSE 0 END) AS complete_count,
    SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) AS failed_count,
    SUM(CASE WHEN status NOT IN ('COMPLETE', 'FAILED') THEN 1 ELSE 0 END) AS active_count,
    MIN(created_at) AS created_at,
    MAX(updated_at) AS updated_at
FROM tasks
GROUP BY workflow_id
ORDER BY MAX(updated_at) DESC
LIMIT ?;
