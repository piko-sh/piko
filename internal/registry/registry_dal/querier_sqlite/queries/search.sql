-- piko.name: FindArtefactIDsByTag
-- piko.command: many
SELECT DISTINCT artefact_id
FROM variant_tag
WHERE tag_key = ? AND tag_value = ?;

-- piko.name: FindArtefactIDsByTagValues
-- piko.command: many
-- ?2 as piko.slice(tag_values)
SELECT DISTINCT artefact_id
FROM variant_tag
WHERE tag_key = ?1 AND tag_value IN (?2);

-- piko.name: FindArtefactByVariantStorageKey
-- piko.command: one
SELECT artefact_id
FROM variant
WHERE storage_key = ?
LIMIT 1;
