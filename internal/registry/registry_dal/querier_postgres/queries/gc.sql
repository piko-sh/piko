-- piko.query(name: AddGCHint, command: exec)
INSERT INTO registry_gc_hint (backend_id, storage_key, created_at)
VALUES ($1, $2, $3);

-- piko.query(name: PopGCHints, command: many)
SELECT id, backend_id, storage_key
FROM registry_gc_hint
ORDER BY id ASC
LIMIT $1;

-- piko.query(name: DeleteGCHints, command: exec)
-- $1 as piko.param(ids, kind: slice)
DELETE FROM registry_gc_hint WHERE id IN ($1);
