-- piko.name: FetchByPriorityStatusesAndActive
-- piko.command: many
-- ?2 as piko.slice(statuses)
SELECT id, status, priority
FROM tasks
WHERE priority >= ?1 AND status IN (?2) AND active = ?3
ORDER BY id ASC;
