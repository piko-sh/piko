-- piko.query(name: FetchByPriorityStatusesAndActive, command: many)
-- ?2 as piko.param(statuses, kind: slice)
SELECT id, status, priority
FROM tasks
WHERE priority >= ?1 AND status IN (?2) AND active = ?3
ORDER BY id ASC;
