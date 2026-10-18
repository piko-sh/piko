-- piko.query(name: UpsertNode, command: exec)
INSERT INTO nodes (instance, node_id, started_at, renewed_at, expires_at)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(instance) DO UPDATE SET
    node_id = excluded.node_id,
    started_at = excluded.started_at,
    renewed_at = excluded.renewed_at,
    expires_at = excluded.expires_at;

-- piko.query(name: RenewNode, command: execrows)
UPDATE nodes
SET renewed_at = ?1, expires_at = ?2
WHERE instance = ?3;

-- piko.query(name: DeleteNode, command: exec)
DELETE FROM nodes WHERE instance = ?1;

-- piko.query(name: ListExpiredNodes, command: many)
SELECT instance FROM nodes
WHERE expires_at < ?1
ORDER BY instance
LIMIT ?2;

-- piko.query(name: TouchOwnedTasks, command: execrows)
UPDATE tasks
SET updated_at = ?1
WHERE owner_instance = ?2 AND status = 'PROCESSING';
