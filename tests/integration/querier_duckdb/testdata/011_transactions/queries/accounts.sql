-- piko.name: GetAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = $1;

-- piko.name: UpdateBalance
-- piko.command: exec
UPDATE accounts SET balance = $1 WHERE id = $2;

-- piko.name: ListAccounts
-- piko.command: many
SELECT id, name, balance FROM accounts ORDER BY id;
