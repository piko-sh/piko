-- piko.query(name: GetLinkedProfileID, command: one)
SELECT (SELECT profile_id FROM user_profiles WHERE user_id = ? LIMIT 1) AS pid
FROM profiles
LIMIT 1;
