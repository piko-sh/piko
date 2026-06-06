-- piko.query(name: GetAccount, command: one)
SELECT id, username, email, active FROM accounts WHERE id = ?;
