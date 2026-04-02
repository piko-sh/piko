-- piko.name: GetUser
-- piko.command: one
SELECT id, email FROM users WHERE email = $1
