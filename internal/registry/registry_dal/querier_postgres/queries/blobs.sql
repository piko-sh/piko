-- piko.name: IncrementBlobRefCount
-- piko.command: one
INSERT INTO registry_blob_reference (storage_key, storage_backend_id, content_hash, size_bytes, mime_type, ref_count, created_at, last_referenced_at)
VALUES ($1, $2, $3, $4, $5, 1, $6, $7)
ON CONFLICT(storage_key) DO UPDATE SET
  ref_count = registry_blob_reference.ref_count + 1,
  last_referenced_at = EXCLUDED.last_referenced_at
RETURNING ref_count;

-- piko.name: DecrementBlobRefCount
-- piko.command: one
UPDATE registry_blob_reference
SET ref_count = ref_count - 1,
    last_referenced_at = $1
WHERE storage_key = $2
RETURNING ref_count;

-- piko.name: GetBlobRefCount
-- piko.command: one
SELECT ref_count FROM registry_blob_reference WHERE storage_key = $1;

-- piko.name: DeleteBlobReferenceIfZero
-- piko.command: exec
DELETE FROM registry_blob_reference WHERE storage_key = $1 AND ref_count = 0;
