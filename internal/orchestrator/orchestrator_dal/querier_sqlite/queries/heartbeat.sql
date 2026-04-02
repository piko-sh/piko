-- piko.name: UpdateTaskHeartbeat
-- piko.command: exec
UPDATE tasks
SET updated_at = ?
WHERE id = ? AND status = 'PROCESSING';

-- piko.name: GetStaleProcessingTaskCount
-- piko.command: one
SELECT COUNT(*) FROM tasks
WHERE status = 'PROCESSING'
AND updated_at < ?;
