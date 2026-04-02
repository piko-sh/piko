-- piko.name: FindArtefactIDsByTag
-- piko.command: many
SELECT DISTINCT artefact_id
FROM registry_variant_tag
WHERE tag_key = $1 AND tag_value = $2;

-- piko.name: FindArtefactIDsByTagValues
-- piko.command: many
-- $2 as piko.slice(tag_values)
SELECT DISTINCT artefact_id
FROM registry_variant_tag
WHERE tag_key = $1 AND tag_value = ANY($2);

-- piko.name: FindArtefactByVariantStorageKey
-- piko.command: one
SELECT artefact_id
FROM registry_variant
WHERE storage_key = $1
LIMIT 1;
