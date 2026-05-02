-- piko.query(name: FetchByStatusesAndPriorities, command: many)
-- ?1 as piko.param(statuses, kind: slice)
-- ?2 as piko.param(priorities, kind: slice)
SELECT id, status, priority
FROM tasks
WHERE status IN (?1) AND priority IN (?2)
ORDER BY id ASC;
