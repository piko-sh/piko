-- piko.name: GetDesiredProfilesForArtefact
-- piko.command: many
SELECT name, capability_name, priority, params_json, tags_json, depends_on_json
FROM desired_profile
WHERE artefact_id = ?;

-- piko.name: GetDesiredProfilesForArtefactIDs
-- piko.command: many
-- ?1 as piko.slice(ids)
SELECT artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json
FROM desired_profile
WHERE artefact_id IN (?1);

-- piko.name: InsertDesiredProfile
-- piko.command: exec
INSERT INTO desired_profile (artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- piko.name: DeleteDesiredProfilesForArtefact
-- piko.command: exec
DELETE FROM desired_profile WHERE artefact_id = ?;
