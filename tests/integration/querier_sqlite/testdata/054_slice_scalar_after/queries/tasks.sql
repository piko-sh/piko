-- piko.query(name: FetchByStatusesAndPriority, command: many)
-- ?1 as piko.param(statuses, kind: slice)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1) AND priority >= ?2
ORDER BY id ASC;

-- piko.query(name: CountByStatusesAndPriority, command: one)
-- ?1 as piko.param(statuses, kind: slice)
SELECT COUNT(*) AS total
FROM tasks
WHERE status IN (?1) AND priority >= ?2;
