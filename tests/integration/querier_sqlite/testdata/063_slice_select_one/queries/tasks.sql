-- piko.query(name: FetchTopByIDsAndStatus, command: one)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, status, priority
FROM tasks
WHERE id IN (?1) AND status = ?2
ORDER BY priority DESC
LIMIT 1;
