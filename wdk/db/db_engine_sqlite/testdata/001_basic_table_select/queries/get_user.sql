-- piko.query(name: GetUser, command: one)
SELECT id, name, email, active FROM users WHERE id = ?
