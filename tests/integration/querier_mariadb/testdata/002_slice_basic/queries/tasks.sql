-- piko.name: FetchByStatuses
-- piko.command: many
-- ?1 as piko.slice(statuses)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1)
ORDER BY priority DESC, id ASC;

-- piko.name: FetchByStatusesAndPriority
-- piko.command: many
-- ?1 as piko.slice(statuses)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1) AND priority >= ?2
ORDER BY id ASC;

-- piko.name: DeleteByIDs
-- piko.command: execrows
-- ?1 as piko.slice(ids)
DELETE FROM tasks WHERE id IN (?1);

-- piko.name: CountByStatuses
-- piko.command: one
-- ?1 as piko.slice(statuses)
SELECT COUNT(*) AS total FROM tasks WHERE status IN (?1);
