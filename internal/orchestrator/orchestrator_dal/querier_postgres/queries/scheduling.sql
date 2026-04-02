-- piko.name: PromoteScheduledTasks
-- piko.command: execrows
UPDATE orchestrator_tasks
SET
    status = 'PENDING',
    updated_at = $1
WHERE
    status = 'SCHEDULED'
    AND execute_at <= $2;

-- piko.name: PendingTaskCount
-- piko.command: one
SELECT COUNT(*) FROM orchestrator_tasks
WHERE status IN ('PENDING', 'SCHEDULED', 'RETRYING');
