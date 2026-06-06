-- piko.query(name: GetAccount, command: one)
SELECT id, name, balance FROM accounts WHERE id = ?;
