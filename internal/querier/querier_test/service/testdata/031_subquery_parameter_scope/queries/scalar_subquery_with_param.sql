-- piko.query(name: GetLinkedProfileID, command: one)
SELECT (SELECT profile_id FROM user_profiles WHERE user_id = $1 LIMIT 1) AS pid
FROM profiles
LIMIT 1;
