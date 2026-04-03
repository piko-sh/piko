-- piko.name: GetWorkflowSummary
-- piko.command: many
-- ?1 as piko.param(max_results)
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

-- piko.name: GetActiveTasks
-- piko.command: many
SELECT id, name, status
FROM tasks
WHERE status NOT IN ('COMPLETE', 'FAILED')
ORDER BY id;

-- piko.name: GetTasksOutsidePriorityRange
-- piko.command: many
-- ?1 as piko.param(min_priority)
-- ?2 as piko.param(max_priority)
SELECT id, name, priority
FROM tasks
WHERE priority NOT BETWEEN ?1 AND ?2
ORDER BY id;

-- piko.name: GetTasksNotMatching
-- piko.command: many
-- ?1 as piko.param(pattern)
SELECT id, name
FROM tasks
WHERE name NOT LIKE ?1
ORDER BY id;
