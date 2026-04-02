-- piko.name: InsertAccount
-- piko.command: exec
INSERT INTO accounts (id, name, balance, status) VALUES (?, ?, ?, ?);

-- piko.name: GetAccount
-- piko.command: one
SELECT id, name, balance, status FROM accounts WHERE id = ?;

-- piko.name: ListActive
-- piko.command: many
SELECT id, name, balance FROM accounts WHERE status = 'active' ORDER BY id;
