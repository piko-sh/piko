-- piko.query(name: GetAccount, command: one)
SELECT id, status, name, email FROM accounts WHERE id = $1
