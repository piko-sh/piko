-- piko.query(name: FetchByIDsAndStatus, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, status, title
FROM tasks
WHERE id IN (?1) AND status = ?2
ORDER BY id ASC;
