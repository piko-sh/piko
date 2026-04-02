-- piko.name: FetchByStatuses
-- piko.command: many
-- $1 as piko.slice(statuses)
SELECT id, status
FROM tasks
WHERE status = ANY($1)
ORDER BY id ASC;
