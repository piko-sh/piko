-- piko.query(name: ShareLockAccount, command: one)
SELECT id, name, balance FROM accounts WHERE id = ? FOR SHARE;
