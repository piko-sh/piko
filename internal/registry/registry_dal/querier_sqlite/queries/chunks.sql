-- piko.query(name: InsertVariantChunk, command: exec)
INSERT INTO variant_chunk (
  artefact_id,
  release_id,
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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
FROM variant_chunk
WHERE artefact_id = ? AND variant_id = ?
ORDER BY sequence_number ASC;

-- piko.query(name: GetChunksForVariants, command: many)
-- ?2 as piko.param(variant_ids, kind: slice)
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

-- piko.query(name: DeleteChunksForArtefact, command: exec)
DELETE FROM variant_chunk
WHERE artefact_id = ? AND release_id = ?;

-- piko.query(name: CountChunksForVariant, command: one)
SELECT COUNT(*) FROM variant_chunk
WHERE artefact_id = ? AND variant_id = ?;

-- piko.query(name: FindArtefactByChunkStorageKey, command: one)
SELECT DISTINCT artefact_id FROM variant_chunk
WHERE storage_key = ? LIMIT 1;
