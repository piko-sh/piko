-- piko.query(name: ListProfilesForUser, command: many)
SELECT id, name, role
FROM profiles
WHERE id IN (SELECT profile_id FROM user_profiles WHERE user_id = ?)
ORDER BY name;
