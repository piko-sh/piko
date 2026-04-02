-- piko.name: UpdateTaskHeartbeat
-- piko.command: exec
UPDATE orchestrator_tasks
SET updated_at = $1
WHERE id = $2 AND status = 'PROCESSING';

-- piko.name: GetStaleProcessingTaskCount
-- piko.command: one
SELECT COUNT(*) FROM orchestrator_tasks
WHERE status = 'PROCESSING'
AND updated_at < $1;
