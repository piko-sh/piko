-- piko.query(name: LockAccount, command: one)
SELECT id, name, balance FROM accounts WHERE id = ? FOR UPDATE;
