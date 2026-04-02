-- piko.name: FetchByIDsAndStatus
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT id, status, title
FROM tasks
WHERE id IN (?1) AND status = ?2
ORDER BY id ASC;
