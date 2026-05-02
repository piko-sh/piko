-- piko.query(name: UpdateStatusByIDs, command: execrows)
-- ?2 as piko.param(ids, kind: slice)
UPDATE tasks SET status = ?1 WHERE id IN (?2);

-- piko.query(name: FetchByPriorityAndStatuses, command: many)
-- ?2 as piko.param(statuses, kind: slice)
SELECT id, status
FROM tasks
WHERE priority = ?1 AND status IN (?2)
ORDER BY id ASC;
