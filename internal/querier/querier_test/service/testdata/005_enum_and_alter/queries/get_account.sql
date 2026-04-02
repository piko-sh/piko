-- piko.name: GetAccount
-- piko.command: one
SELECT id, status, name, email FROM accounts WHERE id = $1
