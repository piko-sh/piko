-- piko.query(name: PromoteScheduledTasks, command: execrows)
UPDATE tasks
SET
    status = 'PENDING',
    updated_at = ?
WHERE
    status = 'SCHEDULED'
    AND execute_at <= ?;

-- piko.query(name: PendingTaskCount, command: one)
SELECT COUNT(*) FROM tasks
WHERE status IN ('PENDING', 'SCHEDULED', 'RETRYING');
