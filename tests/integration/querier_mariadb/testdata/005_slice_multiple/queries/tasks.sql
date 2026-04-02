-- piko.name: FetchByStatusesAndPriorities
-- piko.command: many
-- ?1 as piko.slice(statuses)
-- ?2 as piko.slice(priorities)
SELECT id, status, priority
FROM tasks
WHERE status IN (?1) AND priority IN (?2)
ORDER BY id ASC;
