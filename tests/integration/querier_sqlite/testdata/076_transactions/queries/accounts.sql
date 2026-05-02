-- piko.query(name: GetAccount, command: one)
SELECT id, name, balance FROM accounts WHERE id = ?;

-- piko.query(name: UpdateBalance, command: exec)
UPDATE accounts SET balance = ? WHERE id = ?;

-- piko.query(name: ListAccounts, command: many)
SELECT id, name, balance FROM accounts ORDER BY id;
