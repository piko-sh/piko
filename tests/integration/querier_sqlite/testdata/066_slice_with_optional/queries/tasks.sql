-- piko.query(name: FetchByStatusesWithOptionalPriority, command: many)
-- ?1 as piko.param(statuses, kind: slice)
-- ?2 as piko.param(min_priority, optional: true)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1) AND (?2 IS NULL OR priority >= ?2)
ORDER BY id ASC;
