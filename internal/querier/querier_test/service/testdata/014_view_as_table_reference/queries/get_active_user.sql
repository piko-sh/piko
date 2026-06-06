-- piko.query(name: GetActiveUser, command: one)
SELECT id, name FROM active_users WHERE id = $1
