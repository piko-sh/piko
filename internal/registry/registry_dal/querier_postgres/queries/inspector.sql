-- piko.query(name: ListAllArtefactsWithData, command: many)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact;

-- piko.query(name: ListRecentArtefactsWithData, command: many)
SELECT id, source_path, created_at, updated_at, data_fbs
FROM registry_artefact
ORDER BY updated_at DESC
LIMIT $1;

-- piko.query(name: ListVariantStatusCounts, command: many)
SELECT status, COUNT(*) AS variant_count
FROM registry_variant
GROUP BY status;
