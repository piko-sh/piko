-- piko.query(name: AddGCHint, command: exec)
INSERT INTO gc_hint (backend_id, storage_key, created_at)
VALUES (?, ?, ?);

-- piko.query(name: PopGCHints, command: many)
SELECT id, backend_id, storage_key
FROM gc_hint
ORDER BY id ASC
LIMIT ?;

-- piko.query(name: DeleteGCHints, command: exec)
-- ?1 as piko.param(ids, kind: slice)
DELETE FROM gc_hint WHERE id IN (?1);
