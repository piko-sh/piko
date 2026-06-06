-- piko.query(name: GetVariantsForArtefact, command: many)
SELECT variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM registry_variant
WHERE artefact_id = $1;

-- piko.query(name: GetVariantsForArtefactIDs, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM registry_variant
WHERE artefact_id IN ($1);

-- piko.query(name: InsertVariant, command: exec)
INSERT INTO registry_variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- piko.query(name: DeleteVariantsForArtefact, command: exec)
DELETE FROM registry_variant WHERE artefact_id = $1;

-- piko.query(name: InsertVariantTag, command: exec)
INSERT INTO registry_variant_tag (artefact_id, variant_id, tag_key, tag_value)
VALUES ($1, $2, $3, $4);

-- piko.query(name: DeleteVariantTagsForArtefact, command: exec)
DELETE FROM registry_variant_tag WHERE artefact_id = $1;

-- piko.query(name: GetAllTagsForArtefact, command: many)
SELECT variant_id, tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id = $1;

-- piko.query(name: GetTagsForVariant, command: many)
SELECT tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.query(name: GetTagsForArtefactIDs, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT artefact_id, variant_id, tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id IN ($1);
