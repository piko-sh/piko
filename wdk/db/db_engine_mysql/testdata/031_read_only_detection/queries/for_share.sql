-- piko.name: ShareLockAccount
-- piko.command: one
SELECT id, name, balance FROM accounts WHERE id = ? FOR SHARE;
