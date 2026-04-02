-- piko.name: InsertVariantChunk
-- piko.command: exec
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

-- piko.name: GetChunksForVariant
-- piko.command: many
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

-- piko.name: GetChunksForVariants
-- piko.command: many
-- $2 as piko.slice(variant_ids)
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
WHERE artefact_id = $1 AND variant_id = ANY($2)
ORDER BY artefact_id, variant_id, sequence_number ASC;

-- piko.name: DeleteChunksForVariant
-- piko.command: exec
DELETE FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.name: CountChunksForVariant
-- piko.command: one
SELECT COUNT(*) FROM registry_variant_chunk
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.name: FindArtefactByChunkStorageKey
-- piko.command: one
SELECT DISTINCT artefact_id FROM registry_variant_chunk
WHERE storage_key = $1 LIMIT 1;
