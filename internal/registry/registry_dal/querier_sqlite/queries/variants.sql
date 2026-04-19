-- piko.query(name: GetVariantsForArtefact, command: many)
SELECT variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM variant
WHERE artefact_id = ?;

-- piko.query(name: GetVariantsForArtefactIDs, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM variant
WHERE artefact_id IN (?1);

-- piko.query(name: InsertVariant, command: exec)
INSERT INTO variant (artefact_id, release_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- piko.query(name: DeleteVariantsForArtefact, command: exec)
DELETE FROM variant WHERE artefact_id = ? AND release_id = ?;

-- piko.query(name: InsertVariantTag, command: exec)
INSERT INTO variant_tag (artefact_id, release_id, variant_id, tag_key, tag_value)
VALUES (?, ?, ?, ?, ?);

-- piko.query(name: DeleteVariantTagsForArtefact, command: exec)
DELETE FROM variant_tag WHERE artefact_id = ? AND release_id = ?;

-- piko.query(name: GetAllTagsForArtefact, command: many)
SELECT variant_id, tag_key, tag_value
FROM variant_tag
WHERE artefact_id = ?;

-- piko.query(name: GetTagsForVariant, command: many)
SELECT tag_key, tag_value
FROM variant_tag
WHERE artefact_id = ? AND variant_id = ?;

-- piko.query(name: GetTagsForArtefactIDs, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT artefact_id, variant_id, tag_key, tag_value
FROM variant_tag
WHERE artefact_id IN (?1);
