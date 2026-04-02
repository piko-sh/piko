-- piko.name: LockAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = ? FOR UPDATE;
