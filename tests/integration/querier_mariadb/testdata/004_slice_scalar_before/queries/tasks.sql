-- piko.name: UpdateStatusByIDs
-- piko.command: execrows
-- ?2 as piko.slice(ids)
UPDATE tasks SET status = ?1 WHERE id IN (?2);

-- piko.name: FetchByPriorityAndStatuses
-- piko.command: many
-- ?2 as piko.slice(statuses)
SELECT id, status
FROM tasks
WHERE priority = ?1 AND status IN (?2)
ORDER BY id ASC;
