-- piko.name: HasProfileLink
-- piko.command: one
SELECT id
FROM profiles
WHERE EXISTS (SELECT 1 FROM user_profiles WHERE user_id = $1 AND profile_id = profiles.id);
