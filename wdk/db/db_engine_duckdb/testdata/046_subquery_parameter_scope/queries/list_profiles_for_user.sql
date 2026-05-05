-- piko.name: ListProfilesForUser
-- piko.command: many
SELECT id, name, role
FROM profiles
WHERE id IN (SELECT profile_id FROM user_profiles WHERE user_id = $1)
ORDER BY name;
