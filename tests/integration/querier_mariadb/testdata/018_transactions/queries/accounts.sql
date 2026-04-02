-- piko.name: GetAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = ?;

-- piko.name: UpdateBalance
-- piko.command: exec
UPDATE accounts SET balance = ? WHERE id = ?;

-- piko.name: ListAccounts
-- piko.command: many
SELECT id, name, balance FROM accounts ORDER BY id;
