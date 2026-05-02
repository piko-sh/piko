-- piko.query(name: ListTaskStatusCounts, command: many)
SELECT status, COUNT(*) AS task_count
FROM orchestrator_tasks
GROUP BY status;

-- piko.query(name: ListRecentTasks, command: many)
SELECT
    id, workflow_id, executor, status, priority, attempt,
    last_error, created_at, updated_at
FROM orchestrator_tasks
ORDER BY updated_at DESC
LIMIT $1;

-- piko.query(name: ListWorkflowSummary, command: many)
SELECT
    workflow_id,
    COUNT(*) AS task_count,
    SUM(CASE WHEN status = 'COMPLETE' THEN 1 ELSE 0 END) AS complete_count,
    SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) AS failed_count,
    SUM(CASE WHEN status NOT IN ('COMPLETE', 'FAILED') THEN 1 ELSE 0 END) AS active_count,
    MIN(created_at) AS created_at,
    MAX(updated_at) AS updated_at
FROM orchestrator_tasks
GROUP BY workflow_id
ORDER BY MAX(updated_at) DESC
LIMIT $1;
