-- piko.name: InsertAccount
-- piko.command: one
INSERT INTO accounts (username, status)
VALUES ($1, $2::status_type)
RETURNING id, username, status;

-- piko.name: ListByStatus
-- piko.command: many
SELECT id, username, status
FROM accounts
WHERE status = $1::status_type
ORDER BY id;

-- piko.name: ListAllAccounts
-- piko.command: many
SELECT id, username, status
FROM accounts
ORDER BY id;
