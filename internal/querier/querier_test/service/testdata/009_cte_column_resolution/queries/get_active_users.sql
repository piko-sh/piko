-- piko.name: GetActiveUserNames
-- piko.command: many
WITH active_users AS (SELECT id, name FROM users WHERE active = true)
SELECT active_users.name FROM active_users
