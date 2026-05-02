-- piko.query(name: GetUser, command: one)
SELECT id, email FROM users WHERE email = $1
