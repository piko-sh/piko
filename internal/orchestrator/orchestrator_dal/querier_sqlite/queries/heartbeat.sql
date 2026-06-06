-- piko.query(name: UpdateTaskHeartbeat, command: exec)
UPDATE tasks
SET updated_at = ?
WHERE id = ? AND status = 'PROCESSING';

-- piko.query(name: GetStaleProcessingTaskCount, command: one)
SELECT COUNT(*) FROM tasks
WHERE status = 'PROCESSING'
AND updated_at < ?;
