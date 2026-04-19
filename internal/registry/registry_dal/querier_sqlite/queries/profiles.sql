-- piko.query(name: GetDesiredProfilesForArtefact, command: many)
SELECT name, capability_name, priority, params_json, tags_json, depends_on_json
FROM desired_profile
WHERE artefact_id = ?;

-- piko.query(name: GetDesiredProfilesForArtefactIDs, command: many)
-- ?1 as piko.param(ids, kind: slice)
SELECT artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json
FROM desired_profile
WHERE artefact_id IN (?1);

-- piko.query(name: InsertDesiredProfile, command: exec)
INSERT INTO desired_profile (artefact_id, release_id, name, capability_name, priority, params_json, tags_json, depends_on_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- piko.query(name: DeleteDesiredProfilesForArtefact, command: exec)
DELETE FROM desired_profile WHERE artefact_id = ? AND release_id = ?;
