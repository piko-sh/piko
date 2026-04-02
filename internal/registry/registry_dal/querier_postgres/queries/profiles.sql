-- piko.name: GetDesiredProfilesForArtefact
-- piko.command: many
SELECT name, capability_name, priority, params_json, tags_json, depends_on_json
FROM registry_desired_profile
WHERE artefact_id = $1;

-- piko.name: GetDesiredProfilesForArtefactIDs
-- piko.command: many
-- $1 as piko.slice(ids)
SELECT artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json
FROM registry_desired_profile
WHERE artefact_id = ANY($1);

-- piko.name: InsertDesiredProfile
-- piko.command: exec
INSERT INTO registry_desired_profile (artefact_id, name, capability_name, priority, params_json, tags_json, depends_on_json)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- piko.name: DeleteDesiredProfilesForArtefact
-- piko.command: exec
DELETE FROM registry_desired_profile WHERE artefact_id = $1;
