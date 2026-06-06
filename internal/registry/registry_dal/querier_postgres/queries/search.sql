-- piko.query(name: FindArtefactIDsByTag, command: many)
SELECT DISTINCT artefact_id
FROM registry_variant_tag
WHERE tag_key = $1 AND tag_value = $2;

-- piko.query(name: FindArtefactIDsByTagValues, command: many)
-- $2 as piko.param(tag_values, kind: slice)
SELECT DISTINCT artefact_id
FROM registry_variant_tag
WHERE tag_key = $1 AND tag_value IN ($2);

-- piko.query(name: FindArtefactByVariantStorageKey, command: one)
SELECT artefact_id
FROM registry_variant
WHERE storage_key = $1
LIMIT 1;
