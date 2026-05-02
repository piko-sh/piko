-- piko.query(name: FetchByIDs, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, name FROM items WHERE id IN (?1) ORDER BY id ASC;
