-- piko.name: FetchByStatusesLimited
-- piko.command: many
-- ?1 as piko.slice(statuses)
-- ?2 as piko.limit(page_size)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1)
ORDER BY priority DESC, id ASC
LIMIT ?2;

-- piko.name: CountByStatuses
-- piko.command: one
-- ?1 as piko.slice(statuses)
SELECT COUNT(*) AS total FROM tasks WHERE status IN (?1);
