-- piko.name: GetAccount
-- piko.command: one
SELECT id, username, email, active FROM accounts WHERE id = ?;
