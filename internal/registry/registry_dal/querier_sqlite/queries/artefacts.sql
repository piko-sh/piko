-- piko.query(name: GetArtefact, command: one)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id = ?;

-- piko.query(name: ListAllArtefactIDs, command: many)
SELECT id FROM artefact;

-- piko.query(name: GetMultipleArtefacts, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
WHERE id IN (?1);

-- piko.query(name: UpsertArtefact, command: exec)
INSERT INTO artefact (id, source_path, created_at, updated_at, data_fbs)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  source_path = excluded.source_path,
  updated_at = excluded.updated_at,
  data_fbs = excluded.data_fbs;

-- piko.query(name: DeleteArtefact, command: exec)
DELETE FROM artefact WHERE id = ?;
