-- piko.name: FetchByIDs
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT id, name FROM items WHERE id IN (?1) ORDER BY id ASC;
