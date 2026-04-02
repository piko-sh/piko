-- piko.name: GetUser
-- piko.command: one
SELECT id, name FROM users WHERE id = $1
