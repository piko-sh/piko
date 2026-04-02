-- piko.name: DeleteByStatusAndIDs
-- piko.command: execrows
-- ?2 as piko.slice(ids)
DELETE FROM tasks WHERE status = ?1 AND id IN (?2);

-- piko.name: UpdateStatusByIDs
-- piko.command: exec
-- ?2 as piko.slice(ids)
UPDATE tasks SET status = ?1 WHERE id IN (?2);

-- piko.name: CountNonArchived
-- piko.command: one
SELECT COUNT(*) AS total FROM tasks WHERE status != ?1;
