-- piko.query(name: GetUsersByIDs, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT id, name FROM users WHERE id IN ($1) ORDER BY id;
