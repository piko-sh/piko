-- piko.query(name: GetDesiredProfilesForArtefact, command: many)
SELECT name, capability_name, priority, params_json, tags_json, depends_on_json
FROM registry_desired_profile
WHERE artefact_id = $1;

-- piko.query(name: GetDesiredProfilesForArtefactIDs, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json
FROM registry_desired_profile
WHERE artefact_id IN ($1);

-- piko.query(name: InsertDesiredProfile, command: exec)
INSERT INTO registry_desired_profile (artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- piko.query(name: DeleteDesiredProfilesForArtefact, command: exec)
DELETE FROM registry_desired_profile WHERE artefact_id = $1;
