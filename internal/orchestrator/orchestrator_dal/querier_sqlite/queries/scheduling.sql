-- piko.name: PromoteScheduledTasks
-- piko.command: execrows
UPDATE tasks
SET
    status = 'PENDING',
    updated_at = ?
WHERE
    status = 'SCHEDULED'
    AND execute_at <= ?;

-- piko.name: PendingTaskCount
-- piko.command: one
SELECT COUNT(*) FROM tasks
WHERE status IN ('PENDING', 'SCHEDULED', 'RETRYING');
