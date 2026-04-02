-- piko.name: GetArtefact
-- piko.command: one
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id = $1;

-- piko.name: ListAllArtefactIDs
-- piko.command: many
SELECT id FROM registry_artefact;

-- piko.name: GetMultipleArtefacts
-- piko.command: many
-- $1 as piko.slice(ids)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
WHERE id = ANY($1);

-- piko.name: UpsertArtefact
-- piko.command: exec
INSERT INTO registry_artefact (id, source_path, created_at, updated_at, data_fbs)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(id) DO UPDATE SET
  source_path = EXCLUDED.source_path,
  updated_at = EXCLUDED.updated_at,
  data_fbs = EXCLUDED.data_fbs;

-- piko.name: DeleteArtefact
-- piko.command: exec
DELETE FROM registry_artefact WHERE id = $1;
