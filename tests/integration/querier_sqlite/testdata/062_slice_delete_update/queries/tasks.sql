-- piko.query(name: DeleteByStatusAndIDs, command: execrows)
-- ?2 as piko.param(ids, kind: slice)
DELETE FROM tasks WHERE status = ?1 AND id IN (?2);

-- piko.query(name: UpdateStatusByIDs, command: exec)
-- ?2 as piko.param(ids, kind: slice)
UPDATE tasks SET status = ?1 WHERE id IN (?2);

-- piko.query(name: CountNonArchived, command: one)
SELECT COUNT(*) AS total FROM tasks WHERE status != ?1;
