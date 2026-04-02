-- piko.name: GetArtefact
-- piko.command: one
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id = ?;

-- piko.name: ListAllArtefactIDs
-- piko.command: many
SELECT id FROM artefact;

-- piko.name: GetMultipleArtefacts
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id IN (?1);

-- piko.name: UpsertArtefact
-- piko.command: exec
INSERT INTO artefact (id, source_path, created_at, updated_at, data_fbs)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  source_path = excluded.source_path,
  updated_at = excluded.updated_at,
  data_fbs = excluded.data_fbs;

-- piko.name: DeleteArtefact
-- piko.command: exec
DELETE FROM artefact WHERE id = ?;
