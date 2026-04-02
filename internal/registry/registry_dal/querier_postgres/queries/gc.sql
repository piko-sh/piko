-- piko.name: AddGCHint
-- piko.command: exec
INSERT INTO registry_gc_hint (backend_id, storage_key, created_at)
VALUES ($1, $2, $3);

-- piko.name: PopGCHints
-- piko.command: many
SELECT id, backend_id, storage_key
FROM registry_gc_hint
ORDER BY id ASC
LIMIT $1;

-- piko.name: DeleteGCHints
-- piko.command: exec
-- $1 as piko.slice(ids)
DELETE FROM registry_gc_hint WHERE id = ANY($1);
