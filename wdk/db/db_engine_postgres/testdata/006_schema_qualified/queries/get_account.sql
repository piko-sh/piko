-- piko.name: GetAccount
-- piko.command: one
SELECT id, username, active FROM app.accounts WHERE id = $1;
