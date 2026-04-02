-- piko.name: FetchByStatusesWithOptionalPriority
-- piko.command: many
-- ?1 as piko.slice(statuses)
-- ?2 as piko.optional(min_priority)
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1) AND (?2 IS NULL OR priority >= ?2)
ORDER BY id ASC;
