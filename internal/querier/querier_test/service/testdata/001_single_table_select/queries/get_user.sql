-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email FROM users WHERE id = $1
