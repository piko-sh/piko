-- piko.name: SearchUsers
-- piko.command: many
-- $1 as piko.param(userId)
-- $2 as piko.param(userName)
-- $3 as piko.param(unused)
SELECT id, name FROM users WHERE id = $1 AND name = $2
