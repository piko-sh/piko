-- piko.query(name: FetchByStatuses, command: many)
-- $1 as piko.param(statuses, kind: slice)
SELECT id, status
FROM tasks
WHERE status IN ($1)
ORDER BY id ASC;
