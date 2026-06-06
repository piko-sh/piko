-- piko.query(name: ListAllArtefactsWithData, command: many)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact;

-- piko.query(name: ListRecentArtefactsWithData, command: many)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM artefact
ORDER BY updated_at DESC
LIMIT ?;

-- piko.query(name: ListVariantStatusCounts, command: many)
SELECT status, COUNT(*) AS variant_count
FROM variant
GROUP BY status;
