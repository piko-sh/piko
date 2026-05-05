-- piko.name: ListProfilesForUserQualified
-- piko.command: many
SELECT id, name
FROM profiles
WHERE id IN (SELECT profile_id FROM user_profiles WHERE user_profiles.user_id = $1);
