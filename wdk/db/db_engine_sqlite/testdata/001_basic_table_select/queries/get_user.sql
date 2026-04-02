-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email, active FROM users WHERE id = ?
