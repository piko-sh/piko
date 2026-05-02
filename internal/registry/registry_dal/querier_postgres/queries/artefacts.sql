-- piko.query(name: GetArtefact, command: one)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id = $1;

-- piko.query(name: ListAllArtefactIDs, command: many)
SELECT id FROM registry_artefact;

-- piko.query(name: GetMultipleArtefacts, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id IN ($1);

-- piko.query(name: UpsertArtefact, command: exec)
INSERT INTO registry_artefact (id, source_path, created_at, updated_at, data_fbs)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(id) DO UPDATE SET
  source_path = EXCLUDED.source_path,
  updated_at = EXCLUDED.updated_at,
  data_fbs = EXCLUDED.data_fbs;

-- piko.query(name: DeleteArtefact, command: exec)
DELETE FROM registry_artefact WHERE id = $1;
