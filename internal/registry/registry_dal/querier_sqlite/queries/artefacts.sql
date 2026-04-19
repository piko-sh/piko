-- piko.query(name: GetArtefact, command: many)
SELECT release_id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id = ?
ORDER BY release_id;

-- piko.query(name: GetArtefactForUpdate, command: many)
SELECT release_id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id = ?
ORDER BY release_id;

-- piko.query(name: ListAllArtefactIDs, command: many)
SELECT DISTINCT id FROM artefact;

-- piko.query(name: GetMultipleArtefacts, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, release_id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id IN (?1)
ORDER BY id, release_id;

-- piko.query(name: UpsertArtefact, command: exec)
INSERT INTO artefact (id, release_id, source_path, created_at, updated_at, data_fbs)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id, release_id) DO UPDATE SET
  source_path = excluded.source_path,
  updated_at = excluded.updated_at,
  data_fbs = excluded.data_fbs;

-- piko.query(name: InsertArtefactLayerIfAbsent, command: one, optional: true)
INSERT INTO artefact (id, release_id, source_path, created_at, updated_at, data_fbs)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id, release_id) DO NOTHING
RETURNING id;

-- piko.query(name: DeleteArtefact, command: exec)
DELETE FROM artefact WHERE id = ? AND release_id = '';

-- piko.query(name: DeleteArtefactLayer, command: exec)
DELETE FROM artefact WHERE id = ? AND release_id = ?;

-- piko.query(name: DeleteArtefactLayersForRelease, command: exec)
DELETE FROM artefact WHERE release_id = ?;

-- piko.query(name: ReclaimArtefactLayersForRelease, command: many)
DELETE FROM artefact
WHERE release_id = ?
RETURNING id, release_id, data_fbs;
