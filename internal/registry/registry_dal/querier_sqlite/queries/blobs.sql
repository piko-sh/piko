-- piko.name: IncrementBlobRefCount
-- piko.command: one
INSERT INTO blob_reference (storage_key, storage_backend_id, content_hash, size_bytes, mime_type, ref_count, created_at, last_referenced_at)
VALUES (?, ?, ?, ?, ?, 1, ?, ?)
ON CONFLICT(storage_key) DO UPDATE SET
  ref_count = ref_count + 1,
  last_referenced_at = excluded.last_referenced_at
RETURNING ref_count;

-- piko.name: DecrementBlobRefCount
-- piko.command: one
UPDATE blob_reference
SET ref_count = ref_count - 1,
    last_referenced_at = ?
WHERE storage_key = ?
RETURNING ref_count;

-- piko.name: GetBlobRefCount
-- piko.command: one
SELECT ref_count FROM blob_reference WHERE storage_key = ?;

-- piko.name: DeleteBlobReferenceIfZero
-- piko.command: exec
DELETE FROM blob_reference WHERE storage_key = ? AND ref_count = 0;
