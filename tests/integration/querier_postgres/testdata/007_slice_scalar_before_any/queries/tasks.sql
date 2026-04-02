-- piko.name: FetchByPriorityAndStatuses
-- piko.command: many
-- $2 as piko.slice(statuses)
SELECT id, status, priority, title
FROM tasks
WHERE priority >= $1 AND status = ANY($2)
ORDER BY id ASC;

-- piko.name: CountByStatuses
-- piko.command: one
-- $1 as piko.slice(statuses)
SELECT COUNT(*) AS total FROM tasks WHERE status = ANY($1);
