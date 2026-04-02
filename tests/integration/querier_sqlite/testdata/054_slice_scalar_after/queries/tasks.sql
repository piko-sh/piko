-- piko.name: FetchByStatusesAndPriority
-- piko.command: many
-- ?1 as piko.slice(statuses)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1) AND priority >= ?2
ORDER BY id ASC;

-- piko.name: CountByStatusesAndPriority
-- piko.command: one
-- ?1 as piko.slice(statuses)
SELECT COUNT(*) AS total
FROM tasks
WHERE status IN (?1) AND priority >= ?2;
