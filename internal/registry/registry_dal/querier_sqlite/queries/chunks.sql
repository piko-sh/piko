-- piko.name: InsertVariantChunk
-- piko.command: exec
INSERT INTO variant_chunk (
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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
FROM variant_chunk
WHERE artefact_id = ? AND variant_id = ?
ORDER BY sequence_number ASC;

-- piko.name: GetChunksForVariants
-- piko.command: many
-- ?2 as piko.slice(variant_ids)
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
FROM variant_chunk
WHERE artefact_id = ?1 AND variant_id IN (?2)
ORDER BY artefact_id, variant_id, sequence_number ASC;

-- piko.name: DeleteChunksForVariant
-- piko.command: exec
DELETE FROM variant_chunk
WHERE artefact_id = ? AND variant_id = ?;

-- piko.name: CountChunksForVariant
-- piko.command: one
SELECT COUNT(*) FROM variant_chunk
WHERE artefact_id = ? AND variant_id = ?;

-- piko.name: FindArtefactByChunkStorageKey
-- piko.command: one
SELECT DISTINCT artefact_id FROM variant_chunk
WHERE storage_key = ? LIMIT 1;
