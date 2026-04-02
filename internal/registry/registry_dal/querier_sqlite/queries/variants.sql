-- piko.name: GetVariantsForArtefact
-- piko.command: many
SELECT variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM variant
WHERE artefact_id = ?;

-- piko.name: GetVariantsForArtefactIDs
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM variant
WHERE artefact_id IN (?1);

-- piko.name: InsertVariant
-- piko.command: exec
INSERT INTO variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- piko.name: DeleteVariantsForArtefact
-- piko.command: exec
DELETE FROM variant WHERE artefact_id = ?;

-- piko.name: InsertVariantTag
-- piko.command: exec
INSERT INTO variant_tag (artefact_id, variant_id, tag_key, tag_value)
VALUES (?, ?, ?, ?);

-- piko.name: DeleteVariantTagsForArtefact
-- piko.command: exec
DELETE FROM variant_tag WHERE artefact_id = ?;

-- piko.name: GetAllTagsForArtefact
-- piko.command: many
SELECT variant_id, tag_key, tag_value
FROM variant_tag
WHERE artefact_id = ?;

-- piko.name: GetTagsForVariant
-- piko.command: many
SELECT tag_key, tag_value
FROM variant_tag
WHERE artefact_id = ? AND variant_id = ?;

-- piko.name: GetTagsForArtefactIDs
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT artefact_id, variant_id, tag_key, tag_value
FROM variant_tag
WHERE artefact_id IN (?1);
