-- piko.name: AddGCHint
-- piko.command: exec
INSERT INTO gc_hint (backend_id, storage_key, created_at)
VALUES (?, ?, ?);

-- piko.name: PopGCHints
-- piko.command: many
SELECT id, backend_id, storage_key
FROM gc_hint
ORDER BY id ASC
LIMIT ?;

-- piko.name: DeleteGCHints
-- piko.command: exec
-- ?1 as piko.slice(ids)
DELETE FROM gc_hint WHERE id IN (?1);
