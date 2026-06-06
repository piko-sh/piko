-- piko.query(name: InsertAccount, command: exec)
INSERT INTO accounts (id, name, balance, status) VALUES (?, ?, ?, ?);

-- piko.query(name: GetAccount, command: one)
SELECT id, name, balance, status FROM accounts WHERE id = ?;

-- piko.query(name: ListActive, command: many)
SELECT id, name, balance FROM accounts WHERE status = 'active' ORDER BY id;
