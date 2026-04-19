-- piko.query(name: FindArtefactIDsByTag, command: many)
SELECT DISTINCT artefact_id
FROM variant_tag
WHERE tag_key = ? AND tag_value = ?;

-- piko.query(name: FindArtefactIDsByTagValues, command: many)
-- ?2 as piko.param(tag_values, kind: slice)
SELECT DISTINCT artefact_id
FROM variant_tag
WHERE tag_key = ?1 AND tag_value IN (?2);

-- piko.query(name: FindArtefactByVariantStorageKey, command: one, optional: true)
SELECT artefact_id
FROM variant
WHERE storage_key = ?
LIMIT 1;
