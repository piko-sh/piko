-- piko.query(name: PromoteScheduledTasks, command: execrows)
UPDATE orchestrator_tasks
SET
    status = 'PENDING',
    updated_at = $1
WHERE
    status = 'SCHEDULED'
    AND execute_at <= $2;

-- piko.query(name: PendingTaskCount, command: one)
SELECT COUNT(*) FROM orchestrator_tasks
WHERE status IN ('PENDING', 'SCHEDULED', 'RETRYING');
