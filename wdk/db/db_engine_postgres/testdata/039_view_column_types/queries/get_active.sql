-- piko.query(name: GetActiveUser, command: one)
SELECT id, name, email FROM active_users WHERE id = $1;
