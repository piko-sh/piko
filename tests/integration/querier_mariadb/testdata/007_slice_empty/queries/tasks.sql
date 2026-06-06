-- piko.query(name: FetchByIDs, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, status
FROM tasks
WHERE id IN (?1)
ORDER BY id ASC;

-- piko.query(name: CountByIDs, command: one)
-- ?1 as piko.param(ids, kind: slice)
SELECT COUNT(*) AS total
FROM tasks
WHERE id IN (?1);
