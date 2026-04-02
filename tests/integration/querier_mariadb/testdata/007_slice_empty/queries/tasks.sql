-- piko.name: FetchByIDs
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT id, status
FROM tasks
WHERE id IN (?1)
ORDER BY id ASC;

-- piko.name: CountByIDs
-- piko.command: one
-- ?1 as piko.slice(ids)
SELECT COUNT(*) AS total
FROM tasks
WHERE id IN (?1);
