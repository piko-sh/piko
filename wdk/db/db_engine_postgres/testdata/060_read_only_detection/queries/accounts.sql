-- piko.name: GetAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = $1;

-- piko.name: LockAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = $1 FOR UPDATE;

-- piko.name: CreateAccount
-- piko.command: one
INSERT INTO accounts (name, balance) VALUES ($1, $2) RETURNING id;

-- piko.name: ArchiveAccount
-- piko.command: exec
WITH deleted AS (
    DELETE FROM accounts WHERE id = $1 RETURNING id
)
SELECT id FROM deleted;
