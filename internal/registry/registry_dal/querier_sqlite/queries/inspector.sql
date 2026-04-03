-- piko.name: ListAllArtefactsWithData
-- piko.command: many
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact;

-- piko.name: ListRecentArtefactsWithData
-- piko.command: many
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
ORDER BY updated_at DESC
LIMIT ?;

-- piko.name: ListVariantStatusCounts
-- piko.command: many
SELECT status, COUNT(*) AS variant_count
FROM variant
GROUP BY status;
