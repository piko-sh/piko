-- piko.name: GetVariantsForArtefact
-- piko.command: many
SELECT variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM registry_variant
WHERE artefact_id = $1;

-- piko.name: GetVariantsForArtefactIDs
-- piko.command: many
-- $1 as piko.slice(ids)
SELECT artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at
FROM registry_variant
WHERE artefact_id = ANY($1);

-- piko.name: InsertVariant
-- piko.command: exec
INSERT INTO registry_variant (artefact_id, variant_id, storage_key, storage_backend_id, mime_type, size_bytes, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- piko.name: DeleteVariantsForArtefact
-- piko.command: exec
DELETE FROM registry_variant WHERE artefact_id = $1;

-- piko.name: InsertVariantTag
-- piko.command: exec
INSERT INTO registry_variant_tag (artefact_id, variant_id, tag_key, tag_value)
VALUES ($1, $2, $3, $4);

-- piko.name: DeleteVariantTagsForArtefact
-- piko.command: exec
DELETE FROM registry_variant_tag WHERE artefact_id = $1;

-- piko.name: GetAllTagsForArtefact
-- piko.command: many
SELECT variant_id, tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id = $1;

-- piko.name: GetTagsForVariant
-- piko.command: many
SELECT tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id = $1 AND variant_id = $2;

-- piko.name: GetTagsForArtefactIDs
-- piko.command: many
-- $1 as piko.slice(ids)
SELECT artefact_id, variant_id, tag_key, tag_value
FROM registry_variant_tag
WHERE artefact_id = ANY($1);
