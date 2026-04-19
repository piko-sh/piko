-- piko.query(name: GetArtefact, command: many)
SELECT release_id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id = $1
ORDER BY release_id;

-- piko.query(name: GetArtefactForUpdate, command: many)
SELECT release_id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id = $1
ORDER BY release_id
FOR UPDATE;

-- piko.query(name: ListAllArtefactIDs, command: many)
SELECT DISTINCT id FROM registry_artefact;

-- piko.query(name: GetMultipleArtefacts, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT id, release_id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id IN ($1)
ORDER BY id, release_id;

-- piko.query(name: UpsertArtefact, command: exec)
INSERT INTO registry_artefact (id, release_id, source_path, created_at, updated_at, data_fbs)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT(id, release_id) DO UPDATE SET
  source_path = EXCLUDED.source_path,
  updated_at = EXCLUDED.updated_at,
  data_fbs = EXCLUDED.data_fbs;

-- piko.query(name: InsertArtefactLayerIfAbsent, command: one, optional: true)
INSERT INTO registry_artefact (id, release_id, source_path, created_at, updated_at, data_fbs)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT(id, release_id) DO NOTHING
RETURNING id;

-- piko.query(name: DeleteArtefact, command: exec)
DELETE FROM registry_artefact WHERE id = $1 AND release_id = '';

-- piko.query(name: DeleteArtefactLayer, command: exec)
DELETE FROM registry_artefact WHERE id = $1 AND release_id = $2;

-- piko.query(name: DeleteArtefactLayersForRelease, command: exec)
DELETE FROM registry_artefact WHERE release_id = $1;

-- piko.query(name: ReclaimArtefactLayersForRelease, command: many)
DELETE FROM registry_artefact
WHERE release_id = $1
RETURNING id, release_id, data_fbs;
