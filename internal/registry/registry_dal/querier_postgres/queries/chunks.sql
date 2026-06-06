-- piko.query(name: InsertVariantChunk, command: exec)
INSERT INTO registry_variant_chunk (
  artefact_id,
  variant_id,
  chunk_id,
  storage_key,
  storage_backend_id,
  size_bytes,
  content_hash,
  sequence_number,
  mime_type,
  created_at,
  duration_seconds
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- piko.query(name: GetChunksForVariant, command: many)
SELECT
  chunk_id,
  storage_key,
  storage_backend_id,
  size_bytes,
  content_hash,
  sequence_number,
  mime_type,
  created_at,
  duration_seconds
FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id = $2
ORDER BY sequence_number ASC;

-- piko.query(name: GetChunksForVariants, command: many)
-- $2 as piko.param(variant_ids, kind: slice)
SELECT
  artefact_id,
  variant_id,
  chunk_id,
  storage_key,
  storage_backend_id,
  size_bytes,
  content_hash,
  sequence_number,
  mime_type,
  created_at,
  duration_seconds
FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id IN ($2)
ORDER BY artefact_id, variant_id, sequence_number ASC;

-- piko.query(name: DeleteChunksForVariant, command: exec)
DELETE FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.query(name: CountChunksForVariant, command: one)
SELECT COUNT(*) FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.query(name: FindArtefactByChunkStorageKey, command: one)
SELECT DISTINCT artefact_id FROM registry_variant_chunk
WHERE storage_key = $1 LIMIT 1;
